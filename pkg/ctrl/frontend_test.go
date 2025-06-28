package ctrl

import (
	context "context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	frontendv1alpha1 "github.com/oleksandr-san/k8s-controller/pkg/apis/frontend/v1alpha1"
	testutil "github.com/oleksandr-san/k8s-controller/pkg/testutil"
	"github.com/stretchr/testify/require"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func printTableState(ctx context.Context, c client.Client, ns string, t *testing.T, step string) {
	var pages frontendv1alpha1.FrontendList
	var cms corev1.ConfigMapList
	var deps appsv1.DeploymentList

	c.List(ctx, &pages, client.InNamespace(ns))
	c.List(ctx, &cms, client.InNamespace(ns))
	c.List(ctx, &deps, client.InNamespace(ns))

	t.Logf("\n==== ETCD STATE (%s) ====", step)
	t.Logf("%-15s %-15s %-10s %-10s", "KIND", "NAME", "NAMESPACE", "EXTRA")
	for _, p := range pages.Items {
		t.Logf("%-15s %-15s %-10s contents=%.10s", "Frontend", p.Name, p.Namespace, p.Spec.Contents)
	}
	for _, cm := range cms.Items {
		contents := cm.Data["contents"]
		t.Logf("%-15s %-15s %-10s contents=%.10s", "ConfigMap", cm.Name, cm.Namespace, contents)
	}
	for _, dep := range deps.Items {
		replicas := int32(0)
		if dep.Spec.Replicas != nil {
			replicas = *dep.Spec.Replicas
		}
		t.Logf("%-15s %-15s %-10s replicas=%d", "Deployment", dep.Name, dep.Namespace, replicas)
	}
	if len(pages.Items) == 0 && len(cms.Items) == 0 && len(deps.Items) == 0 {
		t.Logf("<empty>")
	}
}

func TestFrontendReconciler_CreateFlow(t *testing.T) {
	log.SetLogger(zap.New(zap.UseDevMode(true)))

	_, k8sClient, restCfg, cleanup := testutil.StartTestManager(t)
	defer cleanup()

	ctx := context.Background()
	ns := "default"

	// 1. Check if the CRD is present
	extClient, err := apiextensionsclient.NewForConfig(restCfg)
	require.NoError(t, err)
	crd, err := extClient.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, "frontends.frontend.oleksandr-san.io", metav1.GetOptions{})
	require.NoError(t, err, "CRD frontends.frontend.oleksandr-san.io should be present in the cluster")
	require.Equal(t, "frontends.frontend.oleksandr-san.io", crd.Name)

	printTableState(ctx, k8sClient, ns, t, "initial")

	page := &frontendv1alpha1.Frontend{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-page",
			Namespace: ns,
		},
		Spec: frontendv1alpha1.FrontendSpec{
			Contents: "hello world",
			Image:    "nginx:alpine",
			Replicas: 1,
		},
	}
	if err := k8sClient.Create(ctx, page); err != nil {
		t.Fatalf("Failed to create Frontend: %v", err)
	}

	time.Sleep(1 * time.Second) // allow reconcile
	printTableState(ctx, k8sClient, ns, t, "after create")

	// 2. List and check the CR is present
	var pageList frontendv1alpha1.FrontendList
	err = k8sClient.List(ctx, &pageList, client.InNamespace(ns))
	require.NoError(t, err)
	require.NotEmpty(t, pageList.Items, "Should find at least one Frontend")
	found := false
	for _, p := range pageList.Items {
		if p.Name == "test-page" && p.Spec.Contents == "hello world" {
			found = true
		}
	}
	require.True(t, found, "Created Frontend should be present and correct")

	// Update
	page.Spec.Contents = "updated!"
	if err := k8sClient.Update(ctx, page); err != nil {
		t.Fatalf("Failed to update Frontend: %v", err)
	}
	time.Sleep(1 * time.Second)
	printTableState(ctx, k8sClient, ns, t, "after update")

	// Delete
	if err := k8sClient.Delete(ctx, page); err != nil {
		t.Fatalf("Failed to delete Frontend: %v", err)
	}
	time.Sleep(1 * time.Second)
	printTableState(ctx, k8sClient, ns, t, "after delete")
}
