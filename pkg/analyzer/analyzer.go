package analyzer

import (
	"context"
	"fmt"

	rpc "buf.build/gen/go/k8sgpt-ai/k8sgpt/grpc/go/schema/v1/schemav1grpc"
	v1 "buf.build/gen/go/k8sgpt-ai/k8sgpt/protocolbuffers/go/schema/v1"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"k8s.io/client-go/dynamic"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Handler struct {
	rpc.AnalyzerServiceServer
}
type Analyzer struct {
	Handler *Handler
}

func ListPolicyReports() ([]unstructured.Unstructured, []unstructured.Unstructured, error) {
	// Create in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, nil, err
	}

	// Create the dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}

	// Define GroupVersionResources for PolicyReport and ClusterPolicyReport
	policyReportGVR := schema.GroupVersionResource{
		Group:    "wgpolicyk8s.io",
		Version:  "v1alpha2",
		Resource: "policyreports",
	}

	clusterPolicyReportGVR := schema.GroupVersionResource{
		Group:    "wgpolicyk8s.io",
		Version:  "v1alpha2",
		Resource: "clusterpolicyreports",
	}

	// List all PolicyReports in all namespaces
	policyReportsList, err := dynamicClient.Resource(policyReportGVR).Namespace("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, nil, err
	}

	// List all ClusterPolicyReports (cluster-scoped)
	clusterPolicyReportsList, err := dynamicClient.Resource(clusterPolicyReportGVR).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, nil, err
	}

	return policyReportsList.Items, clusterPolicyReportsList.Items, nil
}

func ListAllDeployments() ([]appsv1.Deployment, error) {
	// Create in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	fmt.Println("Attempting to retrieve deployments")
	// List all deployments in all namespaces
	deployments, err := clientset.AppsV1().Deployments("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	} else {
		fmt.Printf("found: %v", deployments)
	}

	return deployments.Items, nil
}

func (a *Handler) Run(context.Context, *v1.AnalyzerRunRequest) (*v1.AnalyzerRunResponse, error) {
	fmt.Println("Running Kyverno custom analyzer")
	// deps, err := ListAllDeployments()
	// if err != nil {
	// 	fmt.Printf("err: %v", err)
	// } else {
	// 	fmt.Printf("found %v", deps)
	// }
	pols, cpols, err := ListPolicyReports()

	if err != nil {
		fmt.Printf("err: %v", err)
	} else {
		fmt.Printf("found %v, %v", pols, cpols)
	}

	// split in ErrorDetail later
	//table := fmt.Sprintf("pols: %v, cpols: %v", pols, cpols)
	table := fmt.Sprintf("pols: %s", "errors fix them") // token limit

	return &v1.AnalyzerRunResponse{
		Result: &v1.Result{
			Name:    "kyverno",
			Details: fmt.Sprintf("Kyverno reports %v", table),
			Error: []*v1.ErrorDetail{
				{
					Text: fmt.Sprintf("Kyverno reports %v", table),
				},
			},
		},
	}, nil
}
