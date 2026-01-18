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

// rigClient implements RigClient for cluster-scoped Rigs
type rigClient struct {
	client dynamic.ResourceInterface
	scheme *runtime.Scheme
}

func (c *rigClient) Get(ctx context.Context, name string) (*gastownv1alpha1.Rig, error) {
	unstr, err := c.client.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return c.unstructuredToRig(unstr)
}

func (c *rigClient) List(ctx context.Context, opts metav1.ListOptions) (*gastownv1alpha1.RigList, error) {
	unstrList, err := c.client.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	rigList := &gastownv1alpha1.RigList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RigList",
			APIVersion: "gastown.gastown.io/v1alpha1",
		},
	}

	for _, item := range unstrList.Items {
		rig, err := c.unstructuredToRig(&item)
		if err != nil {
			return nil, fmt.Errorf("failed to convert rig: %w", err)
		}
		rigList.Items = append(rigList.Items, *rig)
	}

	return rigList, nil
}

func (c *rigClient) Create(ctx context.Context, rig *gastownv1alpha1.Rig) (*gastownv1alpha1.Rig, error) {
	unstr, err := c.rigToUnstructured(rig)
	if err != nil {
		return nil, err
	}
	created, err := c.client.Create(ctx, unstr, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return c.unstructuredToRig(created)
}

func (c *rigClient) Update(ctx context.Context, rig *gastownv1alpha1.Rig) (*gastownv1alpha1.Rig, error) {
	unstr, err := c.rigToUnstructured(rig)
	if err != nil {
		return nil, err
	}
	updated, err := c.client.Update(ctx, unstr, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	return c.unstructuredToRig(updated)
}

func (c *rigClient) UpdateStatus(ctx context.Context, rig *gastownv1alpha1.Rig) (*gastownv1alpha1.Rig, error) {
	unstr, err := c.rigToUnstructured(rig)
	if err != nil {
		return nil, err
	}
	updated, err := c.client.UpdateStatus(ctx, unstr, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	return c.unstructuredToRig(updated)
}

func (c *rigClient) Delete(ctx context.Context, name string) error {
	return c.client.Delete(ctx, name, metav1.DeleteOptions{})
}

func (c *rigClient) unstructuredToRig(obj *unstructured.Unstructured) (*gastownv1alpha1.Rig, error) {
	rig := &gastownv1alpha1.Rig{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, rig)
	if err != nil {
		return nil, fmt.Errorf("failed to convert unstructured to Rig: %w", err)
	}
	return rig, nil
}

func (c *rigClient) rigToUnstructured(rig *gastownv1alpha1.Rig) (*unstructured.Unstructured, error) {
	rig.TypeMeta = metav1.TypeMeta{
		APIVersion: "gastown.gastown.io/v1alpha1",
		Kind:       "Rig",
	}
	objMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(rig)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Rig to unstructured: %w", err)
	}
	return &unstructured.Unstructured{Object: objMap}, nil
}
