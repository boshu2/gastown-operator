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

// polecatClient implements PolecatClient for namespaced Polecats
type polecatClient struct {
	client    dynamic.ResourceInterface
	scheme    *runtime.Scheme
	namespace string
}

func (c *polecatClient) Get(ctx context.Context, name string) (*gastownv1alpha1.Polecat, error) {
	unstr, err := c.client.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return c.unstructuredToPolecat(unstr)
}

func (c *polecatClient) List(ctx context.Context, opts metav1.ListOptions) (*gastownv1alpha1.PolecatList, error) {
	unstrList, err := c.client.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	polecatList := &gastownv1alpha1.PolecatList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PolecatList",
			APIVersion: "gastown.gastown.io/v1alpha1",
		},
	}

	for _, item := range unstrList.Items {
		polecat, err := c.unstructuredToPolecat(&item)
		if err != nil {
			return nil, fmt.Errorf("failed to convert polecat: %w", err)
		}
		polecatList.Items = append(polecatList.Items, *polecat)
	}

	return polecatList, nil
}

func (c *polecatClient) Create(ctx context.Context, polecat *gastownv1alpha1.Polecat) (*gastownv1alpha1.Polecat, error) {
	unstr, err := c.polecatToUnstructured(polecat)
	if err != nil {
		return nil, err
	}
	created, err := c.client.Create(ctx, unstr, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return c.unstructuredToPolecat(created)
}

func (c *polecatClient) Update(ctx context.Context, polecat *gastownv1alpha1.Polecat) (*gastownv1alpha1.Polecat, error) {
	unstr, err := c.polecatToUnstructured(polecat)
	if err != nil {
		return nil, err
	}
	updated, err := c.client.Update(ctx, unstr, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	return c.unstructuredToPolecat(updated)
}

func (c *polecatClient) UpdateStatus(ctx context.Context, polecat *gastownv1alpha1.Polecat) (*gastownv1alpha1.Polecat, error) {
	unstr, err := c.polecatToUnstructured(polecat)
	if err != nil {
		return nil, err
	}
	updated, err := c.client.UpdateStatus(ctx, unstr, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	return c.unstructuredToPolecat(updated)
}

func (c *polecatClient) Delete(ctx context.Context, name string) error {
	return c.client.Delete(ctx, name, metav1.DeleteOptions{})
}

func (c *polecatClient) unstructuredToPolecat(obj *unstructured.Unstructured) (*gastownv1alpha1.Polecat, error) {
	polecat := &gastownv1alpha1.Polecat{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, polecat)
	if err != nil {
		return nil, fmt.Errorf("failed to convert unstructured to Polecat: %w", err)
	}
	return polecat, nil
}

func (c *polecatClient) polecatToUnstructured(polecat *gastownv1alpha1.Polecat) (*unstructured.Unstructured, error) {
	polecat.TypeMeta = metav1.TypeMeta{
		APIVersion: "gastown.gastown.io/v1alpha1",
		Kind:       "Polecat",
	}
	objMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(polecat)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Polecat to unstructured: %w", err)
	}
	return &unstructured.Unstructured{Object: objMap}, nil
}
