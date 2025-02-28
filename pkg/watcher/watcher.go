package watcher

import (
	"context"
	"fmt"
	"log"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
)

type WatchNotifyFunctions struct {
	AddFunc    func(obj *unstructured.Unstructured)
	UpdateFunc func(obj *unstructured.Unstructured)
	DeleteFunc func(obj *unstructured.Unstructured)
}

type WatcherInterface interface {
	Start(notifyF WatchNotifyFunctions, gvr schema.GroupVersionResource, listOptions metav1.ListOptions) error
	Stop()
	Destroy()
}

type Watcher struct {
	preList bool
	client  dynamic.Interface
	watcher watch.Interface
	running bool
}

func NewWatcher(k8sClient dynamic.Interface, preList bool) WatcherInterface {
	return &Watcher{client: k8sClient, watcher: nil, running: false, preList: preList}
}

func (w *Watcher) Start(notifyF WatchNotifyFunctions, gvr schema.GroupVersionResource, listOptions metav1.ListOptions) error {
	if w.watcher != nil {
		return fmt.Errorf("watcher already started")
	}

	// Get a list of current namespaces from the API server
	nameSpaceGvr := schema.GroupVersionResource{
		Group:    "", // The group is empty for core API groups
		Version:  "v1",
		Resource: "namespaces",
	}

	// List the namespaces
	namespaces, err := w.client.Resource(nameSpaceGvr).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	// List of current objects
	resourceVersion := ""

	if w.preList {
		listOptions.Watch = false
		for _, ns := range namespaces.Items {
			list, err := w.client.Resource(gvr).Namespace(ns.GetName()).List(context.Background(), listOptions)
			if err != nil {
				return err
			}
			for i, item := range list.Items {
				if isResourceVersionHigher(item.GetResourceVersion(), resourceVersion) {
					// Update the resourceVersion to the latest
					resourceVersion = item.GetResourceVersion()
					if w.preList {
						notifyF.AddFunc(&item)
					}
					// Make sure the item is scraped by the GC
					list.Items[i] = unstructured.Unstructured{}
				}
			}
			list.Items = nil
			list = nil
		}
	} else {
		resourceVersion = "0"
	}

	// Start the watcher
	listOptions.ResourceVersion = resourceVersion
	watcher, err := w.client.Resource(gvr).Namespace("").Watch(context.Background(), listOptions)
	if err != nil {
		return err
	}
	w.watcher = watcher
	w.running = true
	currentWatcherContext, cancelFunc := context.WithCancel(context.Background())

	// Function to restart the watcher
	restartWatcher := func() {
		if currentWatcherContext != nil && cancelFunc != nil {
			cancelFunc()
		}
		listOptions.ResourceVersion = resourceVersion
		currentWatcherContext, cancelFunc = context.WithCancel(context.Background())
		w.watcher, err = w.client.Resource(gvr).Namespace("").Watch(currentWatcherContext, listOptions)
		if err != nil {
			log.Printf("watcher restart error: %v, on object: %+v", err, gvr)
		}
		watcher = w.watcher
	}

	go func() {
		// Watch for events
		for {
			event, ok := <-watcher.ResultChan()
			if !ok {
				if w.running {
					log.Printf("Watcher channel closed on object %+v", gvr)
					restartWatcher()
					continue
				} else {
					// Stop the watcher
					return
				}
			}
			switch event.Type {
			case watch.Added:
				// Convert the object to unstructured
				addedObject := event.Object.(*unstructured.Unstructured)
				if addedObject == nil {
					log.Printf("watcher error: addedObject is nil")
					continue
				}
				// Update the resourceVersion
				resourceVersion = addedObject.GetResourceVersion()
				notifyF.AddFunc(addedObject)
				addedObject = nil // Make sure the item is scraped by the GC
			case watch.Modified:
				// Convert the object to unstructured
				modifiedObject := event.Object.(*unstructured.Unstructured)
				if modifiedObject == nil {
					log.Printf("watcher error: modifiedObject is nil")
					continue
				}
				// Update the resourceVersion
				resourceVersion = modifiedObject.GetResourceVersion()
				notifyF.UpdateFunc(modifiedObject)
				modifiedObject = nil // Make sure the item is scraped by the GC
			case watch.Deleted:
				// Convert the object to unstructured
				deletedObject := event.Object.(*unstructured.Unstructured)
				if deletedObject == nil {
					log.Printf("watcher error: deletedObject is nil")
					continue
				}
				// Update the resourceVersion
				resourceVersion = deletedObject.GetResourceVersion()
				notifyF.DeleteFunc(deletedObject)
				deletedObject = nil // Make sure the item is scraped by the GC

			case watch.Bookmark:
				// Update the resourceVersion
				bookmarkObject := event.Object.(*unstructured.Unstructured)
				if bookmarkObject == nil {
					log.Printf("watcher error: bookmarkObject is nil")
					continue
				}
				resourceVersion = bookmarkObject.GetResourceVersion()
				bookmarkObject = nil // Make sure the item is scraped by the GC

			case watch.Error:
				// Convert the object to metav1.Status
				watchError := event.Object.(*metav1.Status)
				// Check if the object reason is "Expired" or "Gone" and restart the watcher
				if watchError.Reason == "Expired" || watchError.Reason == "Gone" || watchError.Code == 410 {
					restartWatcher()
					continue
				} else {
					log.Printf("watcher error: %v, on object %+v", event.Object, gvr)
				}
			}
		}
	}()

	return nil
}

func (w *Watcher) Stop() {
	if w.watcher != nil {
		w.running = false
		w.watcher.Stop()
		w.watcher = nil
	}
}

func (w *Watcher) Destroy() {
}

func isResourceVersionHigher(resourceVersion string, currentResourceVersion string) bool {
	// If the currentResourceVersion is empty, return true
	if currentResourceVersion == "" {
		return true
	}

	// Convert the resourceVersion to int64
	resourceVersionInt, err := strconv.ParseInt(resourceVersion, 10, 64)
	if err != nil {
		return false
	}

	// Convert the currentResourceVersion to int64
	currentResourceVersionInt, err := strconv.ParseInt(currentResourceVersion, 10, 64)
	if err != nil {
		return false
	}

	return resourceVersionInt > currentResourceVersionInt
}
