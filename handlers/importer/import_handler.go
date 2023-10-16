package importer

import (
	"github.com/cenkalti/backoff"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type realClock struct{}

func (_ realClock) Now() time.Time {
	return time.Now()
}

type ImportHandler struct {
	objects          []*ImportedObject
	thresholdSeconds int
	clock            backoff.Clock
}

type ImportedObject struct {
	ImportType ImportType
	ObjectKey  client.ObjectKey
	StartedAt  time.Time
}

func NewImportHandler(thresholdSeconds int, clock backoff.Clock) *ImportHandler {
	if nil == clock {
		clock = realClock{}
	}

	return &ImportHandler{
		clock:            clock,
		thresholdSeconds: thresholdSeconds,
	}
}

func (r *ImportedObject) IsMatching(importType ImportType, objectKey client.ObjectKey) bool {
	return importType == r.ImportType && objectKey.String() == r.ObjectKey.String()
}

func (r *ImportedObject) IsExpired(thresholdTime time.Time) bool {
	return r.StartedAt.Before(thresholdTime)
}

func (r *ImportHandler) IsObjectBeingImported(importType ImportType, objectKey client.ObjectKey) bool {
	index := r.GetIndexByContent(importType, objectKey)

	if index < 0 {
		return false
	}

	object := r.Get(index)

	if object.IsExpired(r.clock.Now().Add(time.Duration(-1*r.thresholdSeconds) * time.Second)) {
		r.DeleteByIndex(index)

		return false
	}

	return true
}

func (r *ImportHandler) GetIndexByContent(importType ImportType, objectKey client.ObjectKey) int {
	for k, v := range r.objects {
		if v.IsMatching(importType, objectKey) {
			return k
		}
	}

	return -1
}

func (r *ImportHandler) GetIndex(object *ImportedObject) int {
	for k, v := range r.objects {
		if object == v {
			return k
		}
	}

	return -1
}

func (r *ImportHandler) Get(index int) *ImportedObject {
	return r.objects[index]
}

func (r *ImportHandler) Delete(object *ImportedObject) {
	r.DeleteByIndex(r.GetIndex(object))
}

func (r *ImportHandler) DeleteByIndex(index int) {
	if index < 0 {
		return
	}

	r.objects = append(r.objects[:index], r.objects[index+1:]...)
}

func (r *ImportHandler) Add(object *ImportedObject) {
	r.objects = append(r.objects, object)
}
