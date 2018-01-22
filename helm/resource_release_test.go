package helm

import (
	"fmt"
	"regexp"
	"sync"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"gopkg.in/yaml.v1"
	"k8s.io/helm/pkg/helm"
)

func TestAccResourceRelease_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy,
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigBasic(testReleaseName, testNamespace, testReleaseName, "0.6.2"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.name", testReleaseName),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.namespace", testNamespace),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.status", "DEPLOYED"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.chart", "mariadb"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "0.6.2"),
			),
		}, {
			Config: testAccHelmReleaseConfigBasic(testReleaseName, testNamespace, testReleaseName, "0.6.2"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "0.6.2"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.status", "DEPLOYED"),
			),
		}},
	})
}

func TestAccResourceRelease_concurrent(t *testing.T) {
	var wg sync.WaitGroup

	wg.Add(3)
	for i := 0; i < 3; i++ {
		go func(name string) {
			resource.Test(t, resource.TestCase{
				Providers:    testAccProviders,
				CheckDestroy: testAccCheckHelmReleaseDestroy,
				Steps: []resource.TestStep{{
					Config: testAccHelmReleaseConfigBasic(name, testNamespace, name, "0.6.2"),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr(
							fmt.Sprintf("helm_release.%s", name), "metadata.0.name", name,
						),
					),
				}},
			})

			wg.Done()
		}(fmt.Sprintf("concurrent-%d", i))
	}

	wg.Wait()
}

func TestAccResourceRelease_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy,
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigBasic(testReleaseName, testNamespace, testReleaseName, "0.6.2"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "0.6.2"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.status", "DEPLOYED"),
			),
		}, {
			Config: testAccHelmReleaseConfigBasic(testReleaseName, testNamespace, testReleaseName, "0.6.3"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "2"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "0.6.3"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.status", "DEPLOYED"),
			),
		}},
	})
}

func TestAccResourceRelease_repository(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigRepository(testNamespace, testReleaseName),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.status", "DEPLOYED"),
				resource.TestCheckResourceAttrSet("helm_release.test", "metadata.0.version"),
			),
		}, {
			Config: testAccHelmReleaseConfigRepository(testNamespace, testReleaseName),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.status", "DEPLOYED"),
				resource.TestCheckResourceAttrSet("helm_release.test", "metadata.0.version"),
			),
		}},
	})
}

func TestAccResourceRelease_repository_url(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigRepositoryURL(testNamespace, testReleaseName),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.status", "DEPLOYED"),
				resource.TestCheckResourceAttrSet("helm_release.test", "metadata.0.version"),
			),
		}, {
			Config: testAccHelmReleaseConfigRepositoryURL(testNamespace, testReleaseName),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.status", "DEPLOYED"),
				resource.TestCheckResourceAttrSet("helm_release.test", "metadata.0.version"),
			),
		}},
	})
}

func TestAccResourceRelease_updateAfterFail(t *testing.T) {
	malformed := `
	resource "helm_release" "test" {
	  name        = "malformed"
	  chart       = "stable/nginx-ingress"
	  set {
	      name = "controller.podAnnotations.\"prometheus\\.io/scrape\""
	      value = "true"
	  }
	}
	`

	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy,
		Steps: []resource.TestStep{{
			Config:             malformed,
			ExpectError:        regexp.MustCompile("failed"),
			ExpectNonEmptyPlan: true,
		}, {
			Config: testAccHelmReleaseConfigBasic(testReleaseName, testNamespace, testReleaseName, "0.6.3"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "0.6.3"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.status", "DEPLOYED"),
			),
		}},
	})
}

func testAccHelmReleaseConfigBasic(resource, ns, name, version string) string {
	return fmt.Sprintf(`
		resource "helm_release" "%s" {
 			name      = %q
			namespace = %q
  			chart     = "stable/mariadb"
			version   = %q

			set {
				name = "foo"
				value = "qux"
			}

			set {
				name = "qux.bar"
				value = 1
			}
		}
	`, resource, name, ns, version)
}

func TestGetValues(t *testing.T) {
	d := resourceRelease().Data(nil)
	d.Set("values", `foo: bar`)
	d.Set("set", []interface{}{
		map[string]interface{}{"name": "foo", "value": "qux"},
	})

	values, err := getValues(d)
	if err != nil {
		t.Fatalf("error getValues: %s", err)
		return
	}

	base := map[string]string{}
	err = yaml.Unmarshal([]byte(values), &base)
	if err != nil {
		t.Fatalf("error parsing returned yaml: %s", err)
		return
	}

	if base["foo"] != "qux" {
		t.Fatalf("error merging values, expected %q, got %q", "qux", base["foo"])
	}
}

func testAccHelmReleaseConfigRepository(ns, name string) string {
	return fmt.Sprintf(`
		resource "helm_repository" "incubator" {
			name = "incubator"
			url  = "https://kubernetes-charts-incubator.storage.googleapis.com"
		}

		resource "helm_release" "test" {
 			name       = %q
			namespace  = %q
			repository = "${helm_repository.incubator.metadata.0.name}"
  			chart      = "redis-cache"
		}
	`, name, ns)
}

func testAccHelmReleaseConfigRepositoryURL(ns, name string) string {
	return fmt.Sprintf(`
		resource "helm_release" "test" {
			name       = %q
			namespace  = %q
			repository = "https://kubernetes-charts-incubator.storage.googleapis.com"
			chart      = "redis-cache"
		}
	`, name, ns)
}

func testAccCheckHelmReleaseDestroy(s *terraform.State) error {
	m := testAccProvider.Meta()
	if m == nil {
		return fmt.Errorf("provider not properly initialized")
	}

	client, err := m.(*Meta).GetHelmClient()
	if err != nil {
		return err
	}

	res, err := client.ListReleases(
		helm.ReleaseListNamespace(testNamespace),
	)

	if err != nil {
		return err
	}

	for _, r := range res.Releases {
		if r.Name == testReleaseName {
			return fmt.Errorf("found %q release", testReleaseName)
		}
	}

	if res.Count != 0 {
		return fmt.Errorf("%q namespace should be empty", testNamespace)
	}

	return nil
}
