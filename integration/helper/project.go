package helper

import (
	"testing"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/acorn-io/runtime/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TempProject(t *testing.T, client client.WithWatch) *v1.ProjectInstance {
	t.Helper()
	project := &v1.ProjectInstance{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "acorn-project-",
			Labels: map[string]string{
				"test.acorn.io/project": "true",
				labels.AcornProject:     "true",
			},
		},
		Spec: v1.ProjectInstanceSpec{
			DefaultRegion:    apiv1.LocalRegion,
			SupportedRegions: []string{apiv1.LocalRegion},
		},
	}

	ctx := GetCTX(t)
	err := client.Create(ctx, project)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			t.Fatal(err)
		}
		// Namespace already exists.
		// Will want to get the existing project object to return it
		// and set a cleaning function to remove it after testing.
		t.Logf("Project %s already exists, skipping creation.\n", project.Name)

		if err = client.Get(ctx, k8sclient.ObjectKey{Name: project.Name}, project); err != nil {
			t.Logf("Could not get project %s.\n", project.Name)
			t.Fatal(err)
		}
	}

	// Wait for the project namespace to exist...
	Wait(t, client.Watch, &corev1.NamespaceList{}, func(obj *corev1.Namespace) bool {
		return obj.Name == project.Name
	})

	// Wait for status default region to be set...
	WaitForObject(t, client.Watch, &v1.ProjectInstanceList{}, project, func(obj *v1.ProjectInstance) bool {
		return obj.Status.DefaultRegion == obj.Spec.DefaultRegion
	})

	t.Cleanup(func() {
		err = client.Delete(ctx, project)
		if err != nil {
			t.Logf("Could not delete project %s.\n", project.Name)
			t.Fatal(err)
		}
	})

	return project
}
