package deploy

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"	
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type Deployer struct {
	Config *rest.Config
	Client dynamic.Interface
}

var codecs = serializer.NewCodecFactory(scheme.Scheme)

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

func (d *Deployer) processResource(ctx context.Context, resourceFile []byte) (*unstructured.Unstructured, error) {
	obj, _, err := codecs.UniversalDeserializer().Decode(resourceFile, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to decode resourceFile: %v", err)
	}

	unstructuredObj := &unstructured.Unstructured{}
	unstructuredObj.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, fmt.Errorf("unable to convert obj to unstructured: %v", err)
	}

	if unstructuredObj.GetNamespace() == "" {
		unstructuredObj.SetNamespace("default")
	}

	return unstructuredObj, nil
}

func (d *Deployer) getResourceClient(ctx context.Context, unstructuredObj *unstructured.Unstructured) (dynamic.ResourceInterface, error) {
	c := discovery.NewDiscoveryClientForConfigOrDie(d.Config)
	groupResources, err := restmapper.GetAPIGroupResources(c)
	if err != nil {
		return nil, fmt.Errorf("unable to get API group resources: %v", err)
	}
	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)
	gvk := unstructuredObj.GroupVersionKind()
	mapping, err := mapper.RESTMapping(gvk.GroupKind())
	if err != nil {
		return nil, fmt.Errorf("failed to get REST mapping: %w", err)
	}

	resourceClient := d.Client.Resource(mapping.Resource).Namespace(unstructuredObj.GetNamespace())
	return resourceClient, nil
}
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

func (d *Deployer) AddedFile(ctx context.Context, resourceFile []byte) error {
	unstructuredObj, err := d.processResource(ctx, resourceFile)
	if err != nil {
		return err
	}

	resourceClient, err := d.getResourceClient(ctx, unstructuredObj)
	if err != nil {
		return err
	}

	_, err = resourceClient.Create(ctx, unstructuredObj, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create resource %s: %w", unstructuredObj.GetName(), err)
	}

	log.Printf("Resource created: %s", unstructuredObj.GetName())
	return nil
}
func (d *Deployer) ModifiedFile(ctx context.Context, resourceFile []byte) error {
	unstructuredObj, err := d.processResource(ctx, resourceFile)
	if err != nil {
		return err
	}

	resourceClient, err := d.getResourceClient(ctx, unstructuredObj)
	if err != nil {
		return err
	}

	_, err = resourceClient.Update(ctx, unstructuredObj, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update resource %s: %w", unstructuredObj.GetName(), err)
	}

	log.Printf("Resource updated: %s", unstructuredObj.GetName())
	return nil
}

func (d *Deployer) DeletedFile(ctx context.Context, resourceFile []byte) error {
	unstructuredObj, err := d.processResource(ctx, resourceFile)
	if err != nil {
		return err
	}

	resourceClient, err := d.getResourceClient(ctx, unstructuredObj)
	if err != nil {
		return err
	}

	err = resourceClient.Delete(ctx, unstructuredObj.GetName(), metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete resource %s: %w", unstructuredObj.GetName(), err)
	}

	log.Printf("Resource deleted: %s", unstructuredObj.GetName())
	return nil
}
