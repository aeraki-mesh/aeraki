/*
 * // Copyright Aeraki Authors
 * //
 * // Licensed under the Apache License, Version 2.0 (the "License");
 * // you may not use this file except in compliance with the License.
 * // You may obtain a copy of the License at
 * //
 * //     http://www.apache.org/licenses/LICENSE-2.0
 * //
 * // Unless required by applicable law or agreed to in writing, software
 * // distributed under the License is distributed on an "AS IS" BASIS,
 * // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * // See the License for the specific language governing permissions and
 * // limitations under the License.
 */

package lazyxds_test

import (
	"fmt"
	"github.com/aeraki-mesh/aeraki/test/e2e/lazyxds/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"log"
	"time"
)

const (
	TestNS          = "lazyxds-example"
	lazyServiceName = "lazy-source"
)

var kubeRunner *utils.KubeRunner
var lazySourcePodName string
var normalSourcePodName string

var _ = BeforeSuite(func() {
	var err error
	kubeRunner, err = utils.NewKubeRunnerFromENV()
	if err != nil {
		log.Fatalf("create kube runner failed: %v", err)
	}

	var pod *corev1.Pod

	if err := kubeRunner.CreateNamespace(TestNS, true); err != nil {
		log.Fatalf("create namespace %s failed: %v", TestNS, err)
	}

	By("Apply test source")
	utils.RunCMD(fmt.Sprintf("kubectl -n %s apply -f ../data/source", TestNS))

	By("Apply test services")
	utils.RunCMD(fmt.Sprintf("kubectl apply -n %s -f ../data/services", TestNS))

	// todo wait all pod ok
	time.Sleep(30 * time.Second)
	fmt.Println("pods of istio-system:")
	fmt.Println(utils.RunCMD(fmt.Sprintf("kubectl -n %s get pod", "istio-system")))
	fmt.Println("test services:")
	fmt.Println(utils.RunCMD(fmt.Sprintf("kubectl -n %s get svc", TestNS)))
	fmt.Println("test pods")
	fmt.Println(utils.RunCMD(fmt.Sprintf("kubectl -n %s get pod", TestNS)))
	fmt.Println("test endpoints")
	fmt.Println(utils.RunCMD(fmt.Sprintf("kubectl -n %s get ep", TestNS)))

	pod, err = kubeRunner.GetFirstPodByLabels(TestNS, "app=lazy-source")
	if err != nil {
		log.Fatalf("get pod failed: %v", err)
	}
	lazySourcePodName = pod.Name
	log.Printf("lazyxds source is %s", lazySourcePodName)

	pod, err = kubeRunner.GetFirstPodByLabels(TestNS, "app=normal-source")
	if err != nil {
		log.Fatalf("get pod failed: %v", err)
	}
	normalSourcePodName = pod.Name
	log.Printf("normal source is %s", normalSourcePodName)
})

var _ = AfterSuite(func() {
	By("Delete test namespace")
	utils.RunCMD(fmt.Sprintf("kubectl delete ns %s", TestNS))
})

var _ = Describe("Disable Lazy xDS", func() {
	It("statistics of xDS", func() {
		log.Println(kubeRunner.XDSStatistics(normalSourcePodName, TestNS))
	})
})

var _ = Describe("Enable service Lazy xDS", func() {
	//Context("External Services", func() {
	//	It("Access external HTTP service http://baidu.com, should route to PassthroughCluster", func() {
	//		now := time.Now()
	//		requestID := fmt.Sprintf("request_id=%d", utils.GetRequestID())
	//		out, err := kubeRunner.ExecPod("app", lazySourcePodName, TestNS, fmt.Sprintf("curl -i http://baidu.com?%s", requestID))
	//		if err != nil {
	//			log.Fatalf("kubeRunner.ExecPod failed: %v", err)
	//		}
	//
	//		Expect(out).To(ContainSubstring("HTTP/1.1 200 OK"))
	//
	//		accessLog, err := kubeRunner.GetAccessLog("istio-proxy", lazySourcePodName, TestNS, now, requestID)
	//		if err != nil {
	//			log.Fatalf("kubeRunner.GetAccessLog failed: %v", err)
	//		}
	//		Expect(accessLog).To(ContainSubstring("PassthroughCluster"))
	//
	//		log.Println(kubeRunner.XDSStatistics(lazySourcePodName, TestNS))
	//	})
	//
	//	It("Access external TCP service", func() { // todo need a stable external tcp service
	//	})
	//})

	Context("ServiceEntry", func() {
		It("Access host of ServiceEntry, should access directly", func() {
			now := time.Now()
			tag := fmt.Sprintf("request_id=%d", utils.GetRequestID())

			out, err := kubeRunner.ExecPod("app", lazySourcePodName, TestNS,
				fmt.Sprintf("curl -i http://qq.com:7000/test?%s", tag))
			if err != nil {
				log.Fatalf("kubeRunner.ExecPod failed: %v", err)
			}

			Expect(out).To(ContainSubstring("HTTP/1.1 200 OK"))
			Expect(out).To(ContainSubstring("web-svc1"))

			accessLog, err := kubeRunner.GetAccessLog("istio-proxy", lazySourcePodName, TestNS, now, tag)
			if err != nil {
				log.Fatalf("kubeRunner.GetAccessLog failed: %v", err)
			}
			Expect(accessLog).To(ContainSubstring("outbound|7000||qq.com"))

			log.Println(kubeRunner.XDSStatistics(lazySourcePodName, TestNS))
		})
	})

	Context("Internal service without route rule", func() {
		It("Access TCP service data-svc1, should route to outbound|4000||data-svc1 directly", func() {
			now := time.Now()
			out, err := kubeRunner.ExecPod("app", lazySourcePodName, TestNS, fmt.Sprintf("echo 'quit' | telnet data-svc1.%s.svc.cluster.local 4000", TestNS))
			if err != nil {
				log.Fatalf("kubeRunner.ExecPod failed: %v", err)
			}

			Expect(out).To(ContainSubstring("Connected to data-svc1"))

			svcIP := kubeRunner.GetServiceIP("data-svc1", TestNS)
			accessLog, err := kubeRunner.GetAccessLog("istio-proxy", lazySourcePodName, TestNS, now, fmt.Sprintf("%s:4000", svcIP))
			if err != nil {
				log.Fatalf("kubeRunner.GetAccessLog failed: %v", err)
			}
			Expect(accessLog).To(ContainSubstring("outbound|4000||data-svc1"))

			log.Println(kubeRunner.XDSStatistics(lazySourcePodName, TestNS))
		})

		It("Access HTTP service web-svc1 first time, should route to lazy xds egress", func() {
			now := time.Now()
			tag := fmt.Sprintf("request_id=%d", utils.GetRequestID())
			out, err := kubeRunner.ExecPod("app", lazySourcePodName, TestNS, fmt.Sprintf("curl -i http://web-svc1.%s.svc.cluster.local:7000/test?%s", TestNS, tag))
			if err != nil {
				log.Fatalf("kubeRunner.ExecPod failed: %v", err)
			}

			Expect(out).To(ContainSubstring("HTTP/1.1 200 OK"))
			Expect(out).To(ContainSubstring("web-svc1"))

			accessLog, err := kubeRunner.GetAccessLog("istio-proxy", lazySourcePodName, TestNS, now, tag)
			if err != nil {
				log.Fatalf("kubeRunner.GetAccessLog failed: %v", err)
			}
			Expect(accessLog).To(ContainSubstring("outbound|8080||istio-egressgateway-lazyxds.istio-system.svc.cluster.local"))

			log.Println(kubeRunner.XDSStatistics(lazySourcePodName, TestNS))
		})

		It("Access HTTP service web-svc1 again, should route to target directly", func() {
			now := time.Now()
			tag := fmt.Sprintf("request_id=%d", utils.GetRequestID())

			out, err := kubeRunner.ExecPod("app", lazySourcePodName, TestNS, fmt.Sprintf("curl -i http://web-svc1.%s.svc.cluster.local:7000/test?%s", TestNS, tag))
			if err != nil {
				log.Fatalf("kubeRunner.ExecPod failed: %v", err)
			}

			Expect(out).To(ContainSubstring("HTTP/1.1 200 OK"))
			Expect(out).To(ContainSubstring("web-svc1"))

			accessLog, err := kubeRunner.GetAccessLog("istio-proxy", lazySourcePodName, TestNS, now, tag)
			if err != nil {
				log.Fatalf("kubeRunner.GetAccessLog failed: %v", err)
			}
			Expect(accessLog).To(ContainSubstring(fmt.Sprintf("outbound|7000||web-svc1.%s.svc.cluster.local", TestNS)))

			log.Println(kubeRunner.XDSStatistics(lazySourcePodName, TestNS))
		})
	})

	Context("Internal service with route rule", func() {
		It("Access HTTP service web-svc-multi-version first time, with header user=admin, should route to lazy xds egress, and return v2", func() {
			now := time.Now()
			tag := fmt.Sprintf("request_id=%d", utils.GetRequestID())
			out, err := kubeRunner.ExecPod("app", lazySourcePodName, TestNS,
				fmt.Sprintf("curl -i -H 'user:admin' http://web-svc-multi-version.%s.svc.cluster.local:7001/test?%s", TestNS, tag))
			if err != nil {
				log.Fatalf("kubeRunner.ExecPod failed: %v", err)
			}

			Expect(out).To(ContainSubstring("HTTP/1.1 200 OK"))
			Expect(out).To(ContainSubstring("web-svc-multi-version-v2"))

			accessLog, err := kubeRunner.GetAccessLog("istio-proxy", lazySourcePodName, TestNS, now, tag)
			if err != nil {
				log.Fatalf("kubeRunner.GetAccessLog failed: %v", err)
			}
			Expect(accessLog).To(ContainSubstring("outbound|8080||istio-egressgateway-lazyxds.istio-system.svc.cluster.local"))

			log.Println(kubeRunner.XDSStatistics(lazySourcePodName, TestNS))
		})

		It("Access HTTP service web-svc-multi-version again, without header, should route to target directly, and return v1", func() {
			now := time.Now()
			tag := fmt.Sprintf("request_id=%d", utils.GetRequestID())
			out, err := kubeRunner.ExecPod("app", lazySourcePodName, TestNS,
				fmt.Sprintf("curl -i http://web-svc-multi-version.%s.svc.cluster.local:7001/test?%s", TestNS, tag))
			if err != nil {
				log.Fatalf("kubeRunner.ExecPod failed: %v", err)
			}

			Expect(out).To(ContainSubstring("HTTP/1.1 200 OK"))
			Expect(out).To(ContainSubstring("web-svc-multi-version-v1"))

			accessLog, err := kubeRunner.GetAccessLog("istio-proxy", lazySourcePodName, TestNS, now, tag)
			if err != nil {
				log.Fatalf("kubeRunner.GetAccessLog failed: %v", err)
			}
			Expect(accessLog).To(ContainSubstring(fmt.Sprintf("outbound|7001|v1|web-svc-multi-version.%s.svc.cluster.local", TestNS)))

			log.Println(kubeRunner.XDSStatistics(lazySourcePodName, TestNS))
		})
	})

	Context("Mix HTTP and TCP multi-ports service", func() {
		It("Access TCP service mix-svc:4001, should route to outbound|4001||mix-svc directly", func() {
			now := time.Now()
			out, err := kubeRunner.ExecPod("app", lazySourcePodName, TestNS,
				fmt.Sprintf("echo 'quit' | telnet mix-svc.%s.svc.cluster.local 4001", TestNS))
			if err != nil {
				log.Fatalf("kubeRunner.ExecPod failed: %v", err)
			}

			Expect(out).To(ContainSubstring("Connected to mix-svc"))

			svcIP := kubeRunner.GetServiceIP("mix-svc", TestNS)
			accessLog, err := kubeRunner.GetAccessLog("istio-proxy", lazySourcePodName, TestNS, now, fmt.Sprintf("%s:4001", svcIP))
			if err != nil {
				log.Fatalf("kubeRunner.GetAccessLog failed: %v", err)
			}
			Expect(accessLog).To(ContainSubstring("outbound|4001||mix-svc"))

			log.Println(kubeRunner.XDSStatistics(lazySourcePodName, TestNS))
		})

		It("Access HTTP service mix-svc:7001 first time, should route to target directly", func() {
			now := time.Now()
			tag := fmt.Sprintf("request_id=%d", utils.GetRequestID())
			out, err := kubeRunner.ExecPod("app", lazySourcePodName, TestNS,
				fmt.Sprintf("curl -i http://mix-svc.%s.svc.cluster.local:7001/test?%s", TestNS, tag))
			if err != nil {
				log.Fatalf("kubeRunner.ExecPod failed: %v", err)
			}

			Expect(out).To(ContainSubstring("HTTP/1.1 200 OK"))
			Expect(out).To(ContainSubstring("mix-svc"))

			accessLog, err := kubeRunner.GetAccessLog("istio-proxy", lazySourcePodName, TestNS, now, tag)
			if err != nil {
				log.Fatalf("kubeRunner.GetAccessLog failed: %v", err)
			}
			Expect(accessLog).To(ContainSubstring(fmt.Sprintf("outbound|7001||mix-svc.%s.svc.cluster.local", TestNS)))

			log.Println(kubeRunner.XDSStatistics(lazySourcePodName, TestNS))
		})
	})

	Context("Internal services using virtualService", func() {
		It("Access HTTP service web-svc-related1 first time, with query user=user2, should route to lazy xds egress, and return web-svc-related2", func() {
			now := time.Now()
			tag := fmt.Sprintf("request_id=%d", utils.GetRequestID())
			out, err := kubeRunner.ExecPod("app", lazySourcePodName, TestNS,
				fmt.Sprintf("curl -i http://web-svc-related1.%s.svc.cluster.local:7011/test?user=user2\\&%s", TestNS, tag))
			if err != nil {
				log.Fatalf("kubeRunner.ExecPod failed: %v", err)
			}

			Expect(out).To(ContainSubstring("HTTP/1.1 200 OK"))
			Expect(out).To(ContainSubstring("web-svc-related2"))

			accessLog, err := kubeRunner.GetAccessLog("istio-proxy", lazySourcePodName, TestNS, now, tag)
			if err != nil {
				log.Fatalf("kubeRunner.GetAccessLog failed: %v", err)
			}
			Expect(accessLog).To(ContainSubstring("outbound|8080||istio-egressgateway-lazyxds.istio-system.svc.cluster.local"))

			log.Println(kubeRunner.XDSStatistics(lazySourcePodName, TestNS))
		})

		It("Access HTTP service web-svc-related1, with query user=user3, should route to web-svc-related3 directly", func() {
			now := time.Now()
			tag := fmt.Sprintf("request_id=%d", utils.GetRequestID())
			out, err := kubeRunner.ExecPod("app", lazySourcePodName, TestNS,
				fmt.Sprintf("curl -i http://web-svc-related1.%s.svc.cluster.local:7011/test?user=user3\\&%s", TestNS, tag))
			if err != nil {
				log.Fatalf("kubeRunner.ExecPod failed: %v", err)
			}

			Expect(out).To(ContainSubstring("HTTP/1.1 200 OK"))
			Expect(out).To(ContainSubstring("web-svc-related3"))

			accessLog, err := kubeRunner.GetAccessLog("istio-proxy", lazySourcePodName, TestNS, now, tag)
			if err != nil {
				log.Fatalf("kubeRunner.GetAccessLog failed: %v", err)
			}
			Expect(accessLog).To(ContainSubstring(fmt.Sprintf("outbound|7013||web-svc-related3.%s.svc.cluster.local", TestNS)))

			log.Println(kubeRunner.XDSStatistics(lazySourcePodName, TestNS))
		})

		It("Access HTTP service web-svc-related1, without query params, should route to web-svc-related1 directly", func() {
			now := time.Now()
			tag := fmt.Sprintf("request_id=%d", utils.GetRequestID())
			out, err := kubeRunner.ExecPod("app", lazySourcePodName, TestNS,
				fmt.Sprintf("curl -i http://web-svc-related1.%s.svc.cluster.local:7011/test?%s", TestNS, tag))
			if err != nil {
				log.Fatalf("kubeRunner.ExecPod failed: %v", err)
			}

			Expect(out).To(ContainSubstring("HTTP/1.1 200 OK"))
			Expect(out).To(ContainSubstring("web-svc-related1"))

			accessLog, err := kubeRunner.GetAccessLog("istio-proxy", lazySourcePodName, TestNS, now, tag)
			if err != nil {
				log.Fatalf("kubeRunner.GetAccessLog failed: %v", err)
			}
			Expect(accessLog).To(ContainSubstring(fmt.Sprintf("outbound|7011||web-svc-related1.%s.svc.cluster.local", TestNS)))

			log.Println(kubeRunner.XDSStatistics(lazySourcePodName, TestNS))
		})
	})

	Context("Services unmet lazy xds requirements", func() {
		It("ExternalName type service should always disable lazyxds", func() {
			sidecars := utils.RunCMD(fmt.Sprintf("kubectl -n %s get sidecar -o jsonpath={..metadata.name}", TestNS))

			Expect(sidecars).NotTo(ContainSubstring("external-name-svc-without-selector"))
			Expect(sidecars).NotTo(ContainSubstring("external-name-svc-with-selector"))
		})

		It("Service without selector should always disable lazyxds", func() {
			sidecars := utils.RunCMD(fmt.Sprintf("kubectl -n %s get sidecar -o jsonpath={..metadata.name}", TestNS))

			Expect(sidecars).NotTo(ContainSubstring("lazyxds-selector-less-svc"))
		})

	})

	//Context("Headless Service", func() {
	//	It("Access HTTP service headless-svc first time, should route to lazy xds egress", func() {
	//		now := time.Now()
	//		tag := fmt.Sprintf("request_id=%d", utils.GetRequestID())
	//		out, err := kubeRunner.ExecPod("app", lazySourcePodName, TestNS, fmt.Sprintf("curl -i http://headless-svc.%s.svc.cluster.local:7000/test?%s", TestNS, tag))
	//		if err != nil {
	//			log.Fatalf("kubeRunner.ExecPod failed: %v", err)
	//		}
	//
	//		Expect(out).To(ContainSubstring("HTTP/1.1 200 OK"))
	//		Expect(out).To(ContainSubstring("headless-svc"))
	//
	//		accessLog, err := kubeRunner.GetAccessLog("istio-proxy", lazySourcePodName, TestNS, now, tag)
	//		if err != nil {
	//			log.Fatalf("kubeRunner.GetAccessLog failed: %v", err)
	//		}
	//		Expect(accessLog).To(ContainSubstring("outbound|8080||istio-egressgateway-lazyxds.istio-system.svc.cluster.local"))
	//
	//		log.Println(kubeRunner.XDSStatistics(lazySourcePodName, TestNS))
	//	})
	//
	//	It("Access HTTP service headless-svc using pod dns domain, should access service directly", func() {
	//		pod, _ := kubeRunner.GetFirstPodByLabels(TestNS, "app=headless-svc")
	//		podDomain := strings.Replace(pod.Status.PodIP, ".", "-", -1)
	//
	//		now := time.Now()
	//		tag := fmt.Sprintf("request_id=%d", utils.GetRequestID())
	//
	//		out, err := kubeRunner.ExecPod("app", lazySourcePodName, TestNS, fmt.Sprintf("curl -i http://%s.headless-svc.%s.svc.cluster.local:7000/test?%s", podDomain, TestNS, tag))
	//		if err != nil {
	//			log.Fatalf("kubeRunner.ExecPod failed: %v", err)
	//		}
	//
	//		Expect(out).To(ContainSubstring("HTTP/1.1 200 OK"))
	//		Expect(out).To(ContainSubstring("headless-svc"))
	//
	//		accessLog, err := kubeRunner.GetAccessLog("istio-proxy", lazySourcePodName, TestNS, now, tag)
	//		if err != nil {
	//			log.Fatalf("kubeRunner.GetAccessLog failed: %v", err)
	//		}
	//		Expect(accessLog).To(ContainSubstring(fmt.Sprintf("outbound|7000||headless-svc.%s.svc.cluster.local", TestNS)))
	//
	//		log.Println(kubeRunner.XDSStatistics(lazySourcePodName, TestNS))
	//	})
	//	It("Access HTTP service headless-svc using service dns domain, should access service directly", func() {
	//		now := time.Now()
	//		tag := fmt.Sprintf("request_id=%d", utils.GetRequestID())
	//
	//		out, err := kubeRunner.ExecPod("app", lazySourcePodName, TestNS, fmt.Sprintf("curl -i http://headless-svc.%s.svc.cluster.local:7000/test?%s", TestNS, tag))
	//		if err != nil {
	//			log.Fatalf("kubeRunner.ExecPod failed: %v", err)
	//		}
	//
	//		Expect(out).To(ContainSubstring("HTTP/1.1 200 OK"))
	//		Expect(out).To(ContainSubstring("headless-svc"))
	//
	//		accessLog, err := kubeRunner.GetAccessLog("istio-proxy", lazySourcePodName, TestNS, now, tag)
	//		if err != nil {
	//			log.Fatalf("kubeRunner.GetAccessLog failed: %v", err)
	//		}
	//		Expect(accessLog).To(ContainSubstring(fmt.Sprintf("outbound|7000||headless-svc.%s.svc.cluster.local", TestNS)))
	//
	//		log.Println(kubeRunner.XDSStatistics(lazySourcePodName, TestNS))
	//	})
	//})

	//Context("Disable lazyxds", func() {
	//	It(" access internal HTTP service web-svc2, should access service directly", func() {
	//		utils.RunCMD(fmt.Sprintf("kubectl -n %s annotate svc %s %s=false --overwrite", TestNS, lazyServiceName, config.LazyLoadingAnnotation))
	//		now := time.Now()
	//
	//		tag := fmt.Sprintf("request_id=%d", utils.GetRequestID())
	//
	//		out, err := kubeRunner.ExecPod("app", lazySourcePodName, TestNS,
	//			fmt.Sprintf("curl -i http://web-svc2.%s.svc.cluster.local:7000/test?%s", TestNS, tag))
	//		if err != nil {
	//			log.Fatalf("kubeRunner.ExecPod failed: %v", err)
	//		}
	//
	//		Expect(out).To(ContainSubstring("HTTP/1.1 200 OK"))
	//		Expect(out).To(ContainSubstring("web-svc2"))
	//
	//		accessLog, err := kubeRunner.GetAccessLog("istio-proxy", lazySourcePodName, TestNS, now, tag)
	//		if err != nil {
	//			log.Fatalf("kubeRunner.GetAccessLog failed: %v", err)
	//		}
	//		Expect(accessLog).To(ContainSubstring(fmt.Sprintf("outbound|7000||web-svc2.%s.svc.cluster.local", TestNS)))
	//		log.Println(kubeRunner.XDSStatistics(lazySourcePodName, TestNS))
	//	})
	//
	//})
})
