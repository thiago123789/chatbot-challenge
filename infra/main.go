package main

import (
	"fmt"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/eks"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	networkv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/networking/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		t := true
		vpc, err := ec2.LookupVpc(ctx, &ec2.LookupVpcArgs{Default: &t})
		if err != nil {
			return err
		}

		subnet, err := ec2.GetSubnets(ctx, &ec2.GetSubnetsArgs{
			Filters: []ec2.GetSubnetsFilter{
				{Name: "vpc-id", Values: []string{vpc.Id}},
			},
		})
		if err != nil {
			return err
		}

		eksRole, err := iam.NewRole(ctx, "eks-role", &iam.RoleArgs{
			AssumeRolePolicy: pulumi.String(`{
				"Version": "2012-10-17",
				"Statement": [{
					"Sid": "",
					"Effect": "Allow",
					"Principal": {
						"Service": "eks.amazonaws.com"
					},
					"Action": "sts:AssumeRole"
				}]}`),
		})
		if err != nil {
			return err
		}

		eksPolicies := []string{
			"arn:aws:iam::aws:policy/AmazonEKSServicePolicy",
			"arn:aws:iam::aws:policy/AmazonEKSClusterPolicy",
		}

		for _, policy := range eksPolicies {
			_, err := iam.NewRolePolicyAttachment(ctx, "eks-policy-"+policy, &iam.RolePolicyAttachmentArgs{
				PolicyArn: pulumi.String(policy),
				Role:      eksRole.Name,
			})
			if err != nil {
				return err
			}
		}

		nodeGroupRole, err := iam.NewRole(ctx, "node-group-role", &iam.RoleArgs{
			AssumeRolePolicy: pulumi.String(`{
				"Version": "2012-10-17",
				"Statement": [{
					"Sid": "",
					"Effect": "Allow",
					"Principal": {
						"Service": "ec2.amazonaws.com"
					},
					"Action": "sts:AssumeRole"
				}]
			}`),
		})
		if err != nil {
			return err
		}

		nodeGroupPolicies := []string{
			"arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
			"arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy",
			"arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
		}
		for i, nodeGroupPolicy := range nodeGroupPolicies {
			_, err := iam.NewRolePolicyAttachment(ctx, fmt.Sprintf("ngpa-%d", i), &iam.RolePolicyAttachmentArgs{
				Role:      nodeGroupRole.Name,
				PolicyArn: pulumi.String(nodeGroupPolicy),
			})
			if err != nil {
				return err
			}
		}

		// Create a Security Group that we can use to actually connect to our cluster
		clusterSg, err := ec2.NewSecurityGroup(ctx, "cluster-sg", &ec2.SecurityGroupArgs{
			VpcId: pulumi.String(vpc.Id),
			Egress: ec2.SecurityGroupEgressArray{
				ec2.SecurityGroupEgressArgs{
					Protocol:   pulumi.String("-1"),
					FromPort:   pulumi.Int(0),
					ToPort:     pulumi.Int(0),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
			},
			Ingress: ec2.SecurityGroupIngressArray{
				ec2.SecurityGroupIngressArgs{
					Protocol:   pulumi.String("tcp"),
					FromPort:   pulumi.Int(80),
					ToPort:     pulumi.Int(80),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
			},
		})
		if err != nil {
			return err
		}

		cluster, err := eks.NewCluster(ctx, "mybeacon-cluster", &eks.ClusterArgs{
			RoleArn: eksRole.Arn,
			VpcConfig: &eks.ClusterVpcConfigArgs{
				PublicAccessCidrs: pulumi.StringArray{
					pulumi.String("0.0.0.0/0"),
				},
				SecurityGroupIds: pulumi.StringArray{
					clusterSg.ID().ToStringOutput(),
				},
				SubnetIds: toPulumiStringArray(subnet.Ids),
			},
		})
		if err != nil {
			return err
		}

		nodeGroup, err := eks.NewNodeGroup(ctx, "mybeacon-nodegroup", &eks.NodeGroupArgs{
			ClusterName:   cluster.Name,
			NodeGroupName: pulumi.String("eks-code-challenge-nodegroup"),
			NodeRoleArn:   pulumi.StringInput(nodeGroupRole.Arn),
			SubnetIds:     toPulumiStringArray(subnet.Ids),
			ScalingConfig: &eks.NodeGroupScalingConfigArgs{
				DesiredSize: pulumi.Int(1),
				MaxSize:     pulumi.Int(2),
				MinSize:     pulumi.Int(1),
			},
		})
		if err != nil {
			return err
		}

		ctx.Export("kubeconfig", generateKubeconfig(cluster.Endpoint,
			cluster.CertificateAuthority.Data().Elem(), cluster.Name))

		k8sProvider, err := kubernetes.NewProvider(ctx, "k8sprovider", &kubernetes.ProviderArgs{
			Kubeconfig: generateKubeconfig(cluster.Endpoint,
				cluster.CertificateAuthority.Data().Elem(), cluster.Name),
		}, pulumi.DependsOn([]pulumi.Resource{nodeGroup}))
		if err != nil {
			return err
		}

		appLabelsWorker := pulumi.StringMap{
			"app": pulumi.String("worker-chatbot"),
		}
		appLabelsApi := pulumi.StringMap{
			"app": pulumi.String("api-chatbot"),
		}
		_, err = appsv1.NewDeployment(ctx, "chatbot-worker", &appsv1.DeploymentArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Namespace: pulumi.String("default"),
			},
			Spec: appsv1.DeploymentSpecArgs{
				Selector: &metav1.LabelSelectorArgs{
					MatchLabels: appLabelsWorker,
				},
				Replicas: pulumi.Int(1),
				Template: &corev1.PodTemplateSpecArgs{
					Metadata: &metav1.ObjectMetaArgs{
						Labels: appLabelsWorker,
					},
					Spec: &corev1.PodSpecArgs{
						Containers: corev1.ContainerArray{
							corev1.ContainerArgs{
								Name:  pulumi.String("chatbot-worker"),
								Image: pulumi.String("445567112326.dkr.ecr.us-east-2.amazonaws.com/my-beacon-cbot/worker:latest"),
								Env: corev1.EnvVarArray{
									corev1.EnvVarArgs{
										Name:  pulumi.String("TEMPORAL_HOST_PORT"),
										Value: pulumi.String("temporal-frontend.default.svc.cluster.local:7233"),
									},
									corev1.EnvVarArgs{
										Name:  pulumi.String("TEMPORAL_NAMESPACE"),
										Value: pulumi.String("default"),
									},
								},
							},
						},
					},
				},
			},
		}, pulumi.Provider(k8sProvider))
		if err != nil {
			return err
		}

		_, err = appsv1.NewDeployment(ctx, "chatbot-api", &appsv1.DeploymentArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Namespace: pulumi.String("default"),
			},
			Spec: appsv1.DeploymentSpecArgs{
				Selector: &metav1.LabelSelectorArgs{
					MatchLabels: appLabelsApi,
				},
				Replicas: pulumi.Int(1),
				Template: &corev1.PodTemplateSpecArgs{
					Metadata: &metav1.ObjectMetaArgs{
						Labels: appLabelsApi,
					},
					Spec: &corev1.PodSpecArgs{
						Containers: corev1.ContainerArray{
							corev1.ContainerArgs{
								Name:  pulumi.String("chatbot-api"),
								Image: pulumi.String("445567112326.dkr.ecr.us-east-2.amazonaws.com/my-beacon-cbot/api:latest"),
								Env: corev1.EnvVarArray{
									corev1.EnvVarArgs{
										Name:  pulumi.String("TEMPORAL_HOST_PORT"),
										Value: pulumi.String("temporal-frontend.default.svc.cluster.local:7233"),
									},
									corev1.EnvVarArgs{
										Name:  pulumi.String("TEMPORAL_NAMESPACE"),
										Value: pulumi.String("default"),
									},
								},
							},
						},
					},
				},
			},
		}, pulumi.Provider(k8sProvider))
		if err != nil {
			return err
		}

		_, err = corev1.NewService(ctx, "chatbot-worker", &corev1.ServiceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Namespace: pulumi.String("default"),
				Labels:    appLabelsWorker,
			},
			Spec: &corev1.ServiceSpecArgs{
				Ports: corev1.ServicePortArray{
					corev1.ServicePortArgs{
						Name:       pulumi.String("primary"),
						Port:       pulumi.Int(7239),
						TargetPort: pulumi.Int(7239),
					},
					corev1.ServicePortArgs{
						Name:       pulumi.String("secondary"),
						Port:       pulumi.Int(9090),
						TargetPort: pulumi.Int(9090),
					},
				},
				Selector: appLabelsWorker,
				Type:     pulumi.String("ClusterIP"),
			},
		}, pulumi.Provider(k8sProvider))
		if err != nil {
			return err
		}

		service, err := corev1.NewService(ctx, "chatbot-api", &corev1.ServiceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Namespace: pulumi.String("default"),
				Labels:    appLabelsApi,
				Annotations: pulumi.StringMap{
					"service.beta.kubernetes.io/aws-load-balancer-backend-protocol": pulumi.String("http"),
					"service.beta.kubernetes.io/aws-load-balancer-type":             pulumi.String("alb"),
				},
			},
			Spec: &corev1.ServiceSpecArgs{
				Ports: corev1.ServicePortArray{
					corev1.ServicePortArgs{
						Port:       pulumi.Int(80),
						TargetPort: pulumi.Int(3002),
					},
				},
				Selector: appLabelsApi,
				Type:     pulumi.String("LoadBalancer"),
			},
		}, pulumi.Provider(k8sProvider))
		if err != nil {
			return err
		}

		_, err = networkv1.NewIngress(ctx, "chatbot-api-ingress", &networkv1.IngressArgs{
			Kind: pulumi.String("Ingress"),
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String("chatbot-api-ingress"),
			},
			Spec: &networkv1.IngressSpecArgs{
				Rules: networkv1.IngressRuleArray{
					networkv1.IngressRuleArgs{
						Host: pulumi.String("a8e420573a4fc488ab9ae5e0d0604dcd-150742535.us-east-2.elb.amazonaws.com"),
						Http: &networkv1.HTTPIngressRuleValueArgs{
							Paths: networkv1.HTTPIngressPathArray{
								networkv1.HTTPIngressPathArgs{
									PathType: pulumi.String("Prefix"),
									Path:     pulumi.String("/"),
									Backend: networkv1.IngressBackendArgs{
										Service: networkv1.IngressServiceBackendArgs{
											Name: service.Metadata.Name().Elem(),
											Port: networkv1.ServiceBackendPortArgs{
												Number: pulumi.Int(80),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		})
		if err != nil {
			return err
		}

		ctx.Export("url", service.Status.ApplyT(func(status *corev1.ServiceStatus) *string {
			ingress := status.LoadBalancer.Ingress[0]
			if ingress.Hostname != nil {
				return ingress.Hostname
			}
			return ingress.Ip
		}))

		return nil
	})
}

func generateKubeconfig(clusterEndpoint pulumi.StringOutput, certData pulumi.StringOutput, clusterName pulumi.StringOutput) pulumi.StringOutput {
	return pulumi.Sprintf(`{
        "apiVersion": "v1",
        "clusters": [{
            "cluster": {
                "server": "%s",
                "certificate-authority-data": "%s"
            },
            "name": "kubernetes",
        }],
        "contexts": [{
            "context": {
                "cluster": "kubernetes",
                "user": "aws",
            },
            "name": "aws",
        }],
        "current-context": "aws",
        "kind": "Config",
        "users": [{
            "name": "aws",
            "user": {
                "exec": {
                    "apiVersion": "client.authentication.k8s.io/v1beta1",
                    "command": "aws-iam-authenticator",
                    "args": [
                        "token",
                        "-i",
                        "%s",
                    ],
                },
            },
        }],
    }`, clusterEndpoint, certData, clusterName)
}

func toPulumiStringArray(a []string) pulumi.StringArrayInput {
	var res []pulumi.StringInput
	for _, s := range a {
		res = append(res, pulumi.String(s))
	}
	return pulumi.StringArray(res)
}
