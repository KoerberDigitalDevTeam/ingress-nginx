/*
Copyright 2018 The Kubernetes Authors.

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

package settings

import (
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/parnurzeal/gorequest"

	"k8s.io/ingress-nginx/test/e2e/framework"
)

var _ = framework.IngressNginxDescribe("Global External Auth", func() {
	f := framework.NewDefaultFramework("global-auth-url")

	host := "global-auth-url"

	echoServiceName := "http-svc"

	globalExternalAuthSetting := "global-auth-url"

	fooPath := "/foo"
	barPath := "/bar"

	noAuthSetting := "no-auth-locations"
	noAuthLocations := barPath

	enableGlobalExternalAuthAnnotation := "nginx.ingress.kubernetes.io/enable-global-auth" 

	BeforeEach(func() {
		f.NewEchoDeployment()
		f.NewHttpbinDeployment()
	})

	AfterEach(func() {
	})

	Context("when global external authentication is configured", func() {

		BeforeEach(func() {
			globalExternalAuthURL := fmt.Sprintf("http://httpbin.%s.svc.cluster.local:80/status/401", f.IngressController.Namespace)

			By("Adding an ingress rule for /foo")
			fooIng := framework.NewSingleIngress("foo-ingress", fooPath, host, f.IngressController.Namespace, echoServiceName, 80, nil)
			f.EnsureIngress(fooIng)
			f.WaitForNginxServer(host,
				func(server string) bool {
					return Expect(server).Should(ContainSubstring("location /foo"))
				})

			By("Adding an ingress rule for /bar")
			barIng := framework.NewSingleIngress("bar-ingress", barPath, host, f.IngressController.Namespace, echoServiceName, 80, nil)
			f.EnsureIngress(barIng)
			f.WaitForNginxServer(host,
				func(server string) bool {
					return Expect(server).Should(ContainSubstring("location /bar"))
				})

			By("Adding a global-auth-url to configMap")
			f.UpdateNginxConfigMapData(globalExternalAuthSetting, globalExternalAuthURL)
			f.WaitForNginxServer(host,
				func(server string) bool {
					return Expect(server).Should(ContainSubstring(globalExternalAuthURL))
				})
		})

		It("should return status code 401 when request any protected service", func() {

			By("Sending a request to protected service /foo")
			fooResp, _, _ := gorequest.New().
				Get(f.IngressController.HTTPURL+fooPath).
				Set("Host", host).
				End()
			Expect(fooResp.StatusCode).Should(Equal(http.StatusUnauthorized))

			By("Sending a request to protected service /bar")
			barResp, _, _ := gorequest.New().
				Get(f.IngressController.HTTPURL+barPath).
				Set("Host", host).
				End()
			Expect(barResp.StatusCode).Should(Equal(http.StatusUnauthorized))
		})

		It("should return status code 200 when request whitelisted (via no-auth-locations) service and 401 when request protected service", func() {

			By("Adding a no-auth-locations for /bar to configMap")
			f.UpdateNginxConfigMapData(noAuthSetting, noAuthLocations)

			By("Sending a request to protected service /foo")
			fooResp, _, _ := gorequest.New().
				Get(f.IngressController.HTTPURL+fooPath).
				Set("Host", host).
				End()
			Expect(fooResp.StatusCode).Should(Equal(http.StatusUnauthorized))

			By("Sending a request to whitelisted service /bar")
			barResp, _, _ := gorequest.New().
				Get(f.IngressController.HTTPURL+barPath).
				Set("Host", host).
				End()
			Expect(barResp.StatusCode).Should(Equal(http.StatusOK))
		})

		It("should return status code 200 when request whitelisted (via ingress annotation) service and 401 when request protected service", func() {

			By("Adding an ingress rule for /bar with annotation enable-global-auth = false")
			annotations := map[string]string{
				enableGlobalExternalAuthAnnotation : "false",
			}
			barIng := framework.NewSingleIngress("bar-ingress", barPath, host, f.IngressController.Namespace, echoServiceName, 80, &annotations)
			f.EnsureIngress(barIng)
			f.WaitForNginxServer(host,
				func(server string) bool {
					return Expect(server).Should(ContainSubstring("location /bar"))
				})
	
			By("Sending a request to protected service /foo")
			fooResp, _, _ := gorequest.New().
				Get(f.IngressController.HTTPURL+fooPath).
				Set("Host", host).
				End()
			Expect(fooResp.StatusCode).Should(Equal(http.StatusUnauthorized))
	
			By("Sending a request to whitelisted service /bar")
			barResp, _, _ := gorequest.New().
				Get(f.IngressController.HTTPURL+barPath).
				Set("Host", host).
				End()
			Expect(barResp.StatusCode).Should(Equal(http.StatusOK))
		})

	})

})
