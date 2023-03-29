package deploy

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/google/go-github/v45/github"
	"golang.org/x/oauth2"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/restmapper"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var scheme = runtime.NewScheme()
var codecs = serializer.NewCodecFactory(scheme)

func GetKubernetesClient(ctx context.Context, inCluster bool) (*rest.Config, dynamic.Interface, error) {
	var (
		config *rest.Config
		err    error
	)
	if inCluster {
		config, err = rest.InClusterConfig()
		if err != nil {
			return config, nil, err
		}
	} else {
		kubeConfigPath := filepath.Join(homedir.HomeDir(), ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			return config, nil, err
		}
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	return config, client, nil
}

func GetGitHubClient(ctx context.Context, token string) *github.Client {
	if token == "" {
		log.Printf("New blank client generated")
		return github.NewClient(nil)
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc)
}

func Deploy(ctx context.Context, config *rest.Config, client dynamic.Interface, resourceFile []byte) error {

	obj, gvk, err := codecs.UniversalDeserializer().Decode(resourceFile, nil, nil)
	if err != nil {
		return fmt.Errorf("unable to decode resourceFile: %v", err)
	}

	c := discovery.NewDiscoveryClientForConfigOrDie(config) 
	groupResources, err := restmapper.GetAPIGroupResources(c)
	if err != nil {
		return fmt.Errorf("unable to get API group resources: %v", err)
	}
	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)
	mapping, err := mapper.RESTMapping(gvk.GroupKind())
	if err != nil {
		return fmt.Errorf("failed to get REST mapping: %w", err)
	}

	current := &unstructured.Unstructured{}
	current.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return fmt.Errorf("unable to convert obj to unstructured: %v", err)
	}

	if current.GetNamespace() == "" {
		current.SetNamespace("default")
	}
	resourceClient := client.Resource(mapping.Resource).Namespace(current.GetNamespace())
	_, err = resourceClient.Get(ctx, current.GetName(), metav1.GetOptions{})

	if errors.IsNotFound(err) {
		result, err := resourceClient.Create(context.TODO(), current, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create resource %s: %w", result.GetName(), err)
		}
		log.Printf("Resource created: %s", result.GetName())
		return nil

	} else if err != nil {
		return fmt.Errorf("failed to get resource %s: %w", current.GetName(), err)
	} else if err == nil {
		result, err := resourceClient.Update(context.TODO(), current, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update resource %s: %w", result.GetName(), err)
		} else {
			log.Printf("%s %s is updated", result.GetName(), result.GroupVersionKind().Kind)
		}
	}
	log.Printf("Resource updated: %s", current.GetName())
	return nil
}

