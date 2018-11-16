/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package apimachinery

import (
	"fmt"
	"strings"
	"time"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apiextensions-apiserver/test/integration/fixtures"
	utilversion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubernetes/test/e2e/framework"

	. "github.com/onsi/ginkgo"
)

var crdVersion = utilversion.MustParseSemantic("v1.7.0")
var crdOpenAPIVersion = utilversion.MustParseSemantic("v1.13.0")

var _ = SIGDescribe("CustomResourceDefinition resources", func() {

	f := framework.NewDefaultFramework("custom-resource-definition")

	Context("Simple CustomResourceDefinition", func() {
		/*
			Release : v1.9
			Testname: Custom Resource Definition, create
			Description: Create a API extension client, define a random custom resource definition, create the custom resource. API server MUST be able to create the custom resource.
		*/
		framework.ConformanceIt("creating/deleting custom resource definition objects works ", func() {

			framework.SkipUnlessServerVersionGTE(crdVersion, f.ClientSet.Discovery())

			config, err := framework.LoadConfig()
			if err != nil {
				framework.Failf("failed to load config: %v", err)
			}

			apiExtensionClient, err := clientset.NewForConfig(config)
			if err != nil {
				framework.Failf("failed to initialize apiExtensionClient: %v", err)
			}

			randomDefinition := fixtures.NewRandomNameCustomResourceDefinition(v1beta1.ClusterScoped)

			//create CRD and waits for the resource to be recognized and available.
			randomDefinition, err = fixtures.CreateNewCustomResourceDefinition(randomDefinition, apiExtensionClient, f.DynamicClient)
			if err != nil {
				framework.Failf("failed to create CustomResourceDefinition: %v", err)
			}

			defer func() {
				err = fixtures.DeleteCustomResourceDefinition(randomDefinition, apiExtensionClient)
				if err != nil {
					framework.Failf("failed to delete CustomResourceDefinition: %v", err)
				}
			}()
		})

		It("has OpenAPI spec served with CRD Validation chema", func() {
			framework.SkipUnlessServerVersionGTE(crdOpenAPIVersion, f.ClientSet.Discovery())

			config, err := framework.LoadConfig()
			if err != nil {
				framework.Failf("failed to load config: %v", err)
			}

			apiExtensionClient, err := clientset.NewForConfig(config)
			if err != nil {
				framework.Failf("failed to initialize apiExtensionClient: %v", err)
			}

			randomDefinition := fixtures.NewRandomNameCustomResourceDefinition(v1beta1.ClusterScoped)

			//create CRD and waits for the resource to be recognized and available.
			randomDefinition, err = fixtures.CreateNewCustomResourceDefinition(randomDefinition, apiExtensionClient, f.DynamicClient)
			if err != nil {
				framework.Failf("failed to create CustomResourceDefinition: %v", err)
			}

			// TODO(roycaihw): think about tweaking feature gates in e2e test (is it possible/easy
			// to do so?) and have CRD use top-level/per-version schema
			// Also need to test NamespaceScoped CRDs

			// We use a wait.Poll block here because the kube-aggregator openapi
			// controller takes time to rotate the queue and resync apiextensions-apiserver's spec
			if err := wait.Poll(5*time.Second, 120*time.Second, func() (bool, error) {
				data, err := f.ClientSet.CoreV1().RESTClient().Get().
					AbsPath("/swagger.json").
					DoRaw()

				if err != nil {
					return false, err
				}
				// TODO(roycaihw): verify more Paths and List Definitions, also for multiple versions
				baseDefinition := fmt.Sprintf("%s.%s.%s", randomDefinition.Spec.Group, randomDefinition.Spec.Version, randomDefinition.Spec.Names.Kind)
				basePath := fmt.Sprintf("/apis/%s/%s/%s", randomDefinition.Spec.Group, randomDefinition.Spec.Version, randomDefinition.Spec.Names.Plural)
				return strings.Contains(string(data), basePath) &&
					strings.Contains(string(data), baseDefinition), nil
			}); err != nil {
				framework.Failf("failed to wait for apiserver to serve openapi spec for registered CRD: %v", err)
			}

			defer func() {
				err = fixtures.DeleteCustomResourceDefinition(randomDefinition, apiExtensionClient)
				if err != nil {
					framework.Failf("failed to delete CustomResourceDefinition: %v", err)
				}
			}()
		})

	})
})
