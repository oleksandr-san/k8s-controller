package informer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"

	testutil "github.com/oleksandr-san/k8s-controller/pkg/testutil"
)

func TestStartInformer(t *testing.T) {
	env, _, cleanup := testutil.SetupEnv(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)

	// Patch log to write to a buffer or just rely on test output
	added := make(chan string, 2)

	// Patch event handler for test
	mi, err := NewMultiInformer(
		env.Config,
		30*time.Second,
		[]schema.GroupVersionResource{
			{
				Group:    "apps",
				Version:  "v1",
				Resource: "deployments",
			},
		},
		metav1.NamespaceAll,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	mi.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if d, ok := obj.(metav1.Object); ok {
				added <- d.GetName()
			}
		},
	})

	go func() {
		defer wg.Done()
		mi.Start(ctx)
		<-ctx.Done()
	}()

	// Wait for events
	found := map[string]bool{}
	for range 2 {
		select {
		case name := <-added:
			found[name] = true
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for deployment add events")
		}
	}

	require.True(t, found["sample-deployment-1"])
	require.True(t, found["sample-deployment-2"])

	cancel()
	wg.Wait()

	//t.Log("Sleeping for 5 minutes to allow manual kubectl inspection of envtest cluster...")
	//time.Sleep(5 * time.Minute)
}

func TestGetObjectName(t *testing.T) {
	dep := &metav1.PartialObjectMetadata{}
	dep.SetName("my-deployment")
	name := getObjectName(dep)
	if name != "my-deployment" {
		t.Errorf("expected 'my-deployment', got %q", name)
	}
	name = getObjectName("not-an-object")
	if name != "unknown" {
		t.Errorf("expected 'unknown', got %q", name)
	}
}

func TestStartDeploymentInformer_CoversFunction(t *testing.T) {
	env, _, cleanup := testutil.SetupEnv(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mi, err := NewMultiInformer(
		env.Config,
		30*time.Second,
		[]schema.GroupVersionResource{
			{
				Group:    "apps",
				Version:  "v1",
				Resource: "deployments",
			},
		},
		metav1.NamespaceAll,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Run StartDeploymentInformer in a goroutine
	go func() {
		mi.Start(ctx)
	}()

	// Give the informer some time to start and process events
	time.Sleep(1 * time.Second)
	cancel()
}
