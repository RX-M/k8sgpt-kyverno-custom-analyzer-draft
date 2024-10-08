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
	"k8s.io/client-go/tools/clientcmd"

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
	// config, err := rest.InClusterConfig()
	// if err != nil {
	// 	fmt.Println(err)
	// 	return nil, nil, err
	// }

	config, err := clientcmd.BuildConfigFromFlags("", "/Users/ronaldpetty/.kube/config")

	if err != nil {
		fmt.Println(err)
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
	pols, _, _ := ListPolicyReports()

	// for now, just take first error from pols
	if len(pols) == 0 {
		return &v1.AnalyzerRunResponse{
			Result: &v1.Result{
				Name: "kyverno",
				// Details: fmt.Sprintf("Kyverno looking good")
				// Error: []*v1.ErrorDetail{
				// 	{
				// 		Text: fmt.Sprintf("Kyverno looking good"),
				// 	},
				// },
			},
		}, nil
	}

	var x any
	x = pols[0].Object["results"]
	fmt.Printf("%T\n", x)
	fmt.Println(x)
	convertedData, ok := x.([]interface{})
	var table string
	if !ok {
		fmt.Println("Data is not of the expected type")
	} else {
		for _, item := range convertedData {
			if itemMap, ok := item.(map[string]interface{}); ok {
				// Now you can access the map's fields
				fmt.Println("Message:", itemMap["message"])
				fmt.Println("Policy:", itemMap["policy"])
				fmt.Println("Result:", itemMap["result"])
				if itemMap["result"] == "fail" {
					table = fmt.Sprintf("This policy: %s caused this result: %s; here is the message: %s", itemMap["policy"], itemMap["result"], itemMap["message"])
				}
				fmt.Println("Rule:", itemMap["rule"])
				fmt.Println("Scored:", itemMap["scored"])
			}
		}
	}

	// polsJsonData, err := json.Marshal(pols)
	// if err != nil {
	// 	fmt.Println("Error:", err)
	// 	return nil, err
	// } else {
	// 	fmt.Printf("found %s", polsJsonData)
	// }

	// cpolsJsonData, err := json.Marshal(cpols)
	// if err != nil {
	// 	fmt.Println("Error:", err)
	// 	return nil, err
	// } else {
	// 	fmt.Printf("found %s", cpolsJsonData)
	// }

	// if err != nil {
	// 	fmt.Printf("err: %v", err)
	// } else {
	// 	fmt.Printf("found %v, %v", pols, cpols)
	// }

	// split in ErrorDetail later
	//table := fmt.Sprintf("pols: %s, cpols: %s", polsJsonData, cpolsJsonData)[:1000]
	// data := `{"message":"validation error: label 'team' is required. rule check-team failed at path /metadata/labels/team/","policy":"require-labels","source":"kyverno"}`
	// table := fmt.Sprintf("pols: %s", data) // token limit

	return &v1.AnalyzerRunResponse{
		Result: &v1.Result{
			Name:    "kyverno",
			Details: fmt.Sprintf("Kyverno reports %s", table),
			Error: []*v1.ErrorDetail{
				{
					Text: fmt.Sprintf("Kyverno reports %s", table),
				},
			},
		},
	}, nil
}
