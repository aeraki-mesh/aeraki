// Copyright Aeraki Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	redisv1alpha1 "github.com/aeraki-framework/aeraki/client-go/pkg/apis/redis/v1alpha1"
	versioned "github.com/aeraki-framework/aeraki/client-go/pkg/clientset/versioned"
	internalinterfaces "github.com/aeraki-framework/aeraki/client-go/pkg/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/aeraki-framework/aeraki/client-go/pkg/listers/redis/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// RedisServiceInformer provides access to a shared informer and lister for
// RedisServices.
type RedisServiceInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.RedisServiceLister
}

type redisServiceInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewRedisServiceInformer constructs a new informer for RedisService type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewRedisServiceInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredRedisServiceInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredRedisServiceInformer constructs a new informer for RedisService type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredRedisServiceInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.RedisV1alpha1().RedisServices(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.RedisV1alpha1().RedisServices(namespace).Watch(context.TODO(), options)
			},
		},
		&redisv1alpha1.RedisService{},
		resyncPeriod,
		indexers,
	)
}

func (f *redisServiceInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredRedisServiceInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *redisServiceInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&redisv1alpha1.RedisService{}, f.defaultInformer)
}

func (f *redisServiceInformer) Lister() v1alpha1.RedisServiceLister {
	return v1alpha1.NewRedisServiceLister(f.Informer().GetIndexer())
}
