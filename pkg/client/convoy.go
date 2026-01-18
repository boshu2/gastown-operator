/*
Copyright 2026.

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

package client

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"

	gastownv1alpha1 "github.com/org/gastown-operator/api/v1alpha1"
)

// convoyClient implements ConvoyClient for namespaced Convoys
type convoyClient struct {
	client    dynamic.ResourceInterface
	scheme    *runtime.Scheme
	namespace string
}

func (c *convoyClient) Get(ctx context.Context, name string) (*gastownv1alpha1.Convoy, error) {
	unstr, err := c.client.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return c.unstructuredToConvoy(unstr)
}

func (c *convoyClient) List(ctx context.Context, opts metav1.ListOptions) (*gastownv1alpha1.ConvoyList, error) {
	unstrList, err := c.client.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	convoyList := &gastownv1alpha1.ConvoyList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConvoyList",
			APIVersion: "gastown.gastown.io/v1alpha1",
		},
	}

	for _, item := range unstrList.Items {
		convoy, err := c.unstructuredToConvoy(&item)
		if err != nil {
			return nil, fmt.Errorf("failed to convert convoy: %w", err)
		}
		convoyList.Items = append(convoyList.Items, *convoy)
	}

	return convoyList, nil
}

func (c *convoyClient) Create(ctx context.Context, convoy *gastownv1alpha1.Convoy) (*gastownv1alpha1.Convoy, error) {
	unstr, err := c.convoyToUnstructured(convoy)
	if err != nil {
		return nil, err
	}
	created, err := c.client.Create(ctx, unstr, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return c.unstructuredToConvoy(created)
}

func (c *convoyClient) Update(ctx context.Context, convoy *gastownv1alpha1.Convoy) (*gastownv1alpha1.Convoy, error) {
	unstr, err := c.convoyToUnstructured(convoy)
	if err != nil {
		return nil, err
	}
	updated, err := c.client.Update(ctx, unstr, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	return c.unstructuredToConvoy(updated)
}

func (c *convoyClient) UpdateStatus(ctx context.Context, convoy *gastownv1alpha1.Convoy) (*gastownv1alpha1.Convoy, error) {
	unstr, err := c.convoyToUnstructured(convoy)
	if err != nil {
		return nil, err
	}
	updated, err := c.client.UpdateStatus(ctx, unstr, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	return c.unstructuredToConvoy(updated)
}

func (c *convoyClient) Delete(ctx context.Context, name string) error {
	return c.client.Delete(ctx, name, metav1.DeleteOptions{})
}

func (c *convoyClient) unstructuredToConvoy(obj *unstructured.Unstructured) (*gastownv1alpha1.Convoy, error) {
	convoy := &gastownv1alpha1.Convoy{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, convoy)
	if err != nil {
		return nil, fmt.Errorf("failed to convert unstructured to Convoy: %w", err)
	}
	return convoy, nil
}

func (c *convoyClient) convoyToUnstructured(convoy *gastownv1alpha1.Convoy) (*unstructured.Unstructured, error) {
	convoy.TypeMeta = metav1.TypeMeta{
		APIVersion: "gastown.gastown.io/v1alpha1",
		Kind:       "Convoy",
	}
	objMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(convoy)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Convoy to unstructured: %w", err)
	}
	return &unstructured.Unstructured{Object: objMap}, nil
}
