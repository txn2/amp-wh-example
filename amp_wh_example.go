package amp_wh_example

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Config configures the API
type Config struct {
	Log *zap.Logger
}

// Api
type Api struct {
	*Config
}

// PatchOperation
// see: http://jsonpatch.com/
type PatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

// NewApi
func NewApi(cfg *Config) (*Api, error) {
	a := &Api{Config: cfg}

	// default logger if none specified
	if a.Log == nil {
		zapCfg := zap.NewProductionConfig()
		logger, err := zapCfg.Build()
		if err != nil {
			fmt.Print(err.Error())
			os.Exit(1)
		}

		a.Log = logger
	}

	return a, nil
}

// OkHandler
func (a *Api) OkHandler(version string, mode string, service string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"version": version, "mode": mode, "service": service})
	}
}

// MutatePodHandler
func (a *Api) MutatePodHandler() gin.HandlerFunc {
	return func(c *gin.Context) {

		a.Log.Info("MutatePodHandler")

		rs, err := c.GetRawData()
		if err != nil {
			a.Log.Error("unable to get request body.", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"message": "unable to get request body",
				"error":   err.Error(),
			})
			return
		}

		pod := &corev1.Pod{}
		err = json.Unmarshal(rs, pod)
		if err != nil {
			a.Log.Error("unable to Unmarshal request body into Pod.", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"message": "unable to Unmarshal request into Pod",
				"error":   err.Error(),
			})
			return
		}

		// get patch operation
		po, err := a.MutatePod(*pod)
		if err != nil {
			a.Log.Error("MutatePod failed", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusOK, po)
	}
}

// MutatePod
func (a *Api) MutatePod(pod corev1.Pod) ([]PatchOperation, error) {
	po := make([]PatchOperation, 0)

	logInfo := []zap.Field{
		zap.String("PodName", pod.Name),
		zap.String("Namespace", pod.Namespace),
	}

	a.Log.Info("MutatePod got Pod and checking annotation condition.", logInfo...)

	// EXAMPLE: only mutate if the Pod is asking for it
	if pod.Annotations["amp.txn2.com/example"] == "mutate" {
		a.Log.Info("Annotation condition add_init_container. Building PatchOperation")
		po = append(
			po,
			[]PatchOperation{
				// add initContainer before index 1
				{
					Op:   "add",
					Path: "/spec/initContainers/1",
					Value: corev1.Container{
						Name:  "added-init-container-a",
						Image: "alpine:3.12.0",
					},
				},
				// add initContainer before index 2
				{
					Op:   "add",
					Path: "/spec/initContainers/2",
					Value: corev1.Container{
						Name:  "added-init-container-b",
						Image: "alpine:3.12.0",
						Env: []corev1.EnvVar{
							{
								Name:  "SOME_ENV_VAR",
								Value: "amp-example-webhook",
							},
						},
					},
				},
				// remove annotation
				{
					Op: "remove",
					// ~1 == / in key
					Path: "/metadata/annotations/amp.txn2.com~1delete-me",
				},
				// add annotation
				{
					Op:    "add",
					Path:  "/metadata/annotations/amp.txn2.com~1added-annotation",
					Value: "Added by mutation",
				},
				// replace label
				{
					Op:    "replace",
					Path:  "/metadata/labels/mutated",
					Value: "true",
				},
				// add environment variable to container 0 first-existing-container
				{
					Op:   "add",
					Path: "/spec/containers/0/env/-",
					Value: corev1.EnvVar{
						Name:  "ADDED_ENV_VAR",
						Value: "Added by mutation",
					},
				},
				// add a volume
				{
					Op:   "add",
					Path: "/spec/volumes/-",
					Value: corev1.Volume{
						Name: "mutation-added-vol",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
				// add volumeMount to new volume in second container
				{
					Op:   "add",
					Path: "/spec/containers/1/volumeMounts/-",
					Value: corev1.VolumeMount{
						Name:      "mutation-added-vol",
						MountPath: "/mutation-vol",
					},
				},
				// add an init container populating mutation added volume
				{
					Op:   "add",
					Path: "/spec/initContainers/-",
					Value: corev1.Container{
						Name:  "added-init-container-vol-pop",
						Image: "alpine:3.12.0",
						Command: []string{
							"sh",
							"-c",
							"echo $SOME_INIT_VAR > /mutation-vol/test.txt",
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "mutation-added-vol",
								MountPath: "/mutation-vol",
							},
						},
						Env: []corev1.EnvVar{
							{
								Name:  "SOME_INIT_VAR",
								Value: "the value of SOME_INIT_VAR",
							},
						},
					},
				},
				// add an ephemeral-storage resource request the first container
				{
					Op:    "add",
					Path:  "/spec/containers/0/resources/requests/ephemeral-storage",
					Value: "1G",
				},
				// add an ephemeral-storage resource limit the first container
				{
					Op:    "add",
					Path:  "/spec/containers/0/resources/limits/ephemeral-storage",
					Value: "12G",
				},
				// add resource requests and limits to second container
				{
					Op:   "add",
					Path: "/spec/containers/1/resources",
					Value: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("100m"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("500m"),
						},
					},
				},
			}...,
		)
	}

	return po, nil
}
