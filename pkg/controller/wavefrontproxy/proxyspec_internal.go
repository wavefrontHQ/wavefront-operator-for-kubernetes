package wavefrontproxy

import (
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	wfv1 "github.com/wavefronthq/wavefront-operator/pkg/apis/wavefront/v1alpha1"
	"github.com/wavefronthq/wavefront-operator/pkg/controller/util"
	corev1 "k8s.io/api/core/v1"
)

const (
	defaultImage = "wavefronthq/proxy:latest"

	defaultMetricPort = 2878

	defaultImagePullPolicy = corev1.PullIfNotPresent

	deafultPreprocessorMountPath = "/etc/wavefront/wavefront-proxy/preprocessor"

	defaultAdvancedWavefrontConfPath = "/etc/wavefront/wavefront-proxy/conf"
)

// Internal Wavefront Proxy struct contains the actual desired instance plus other runtime objects
// as first class spec configs used during create/update of Container deployments/pod/services.
// This provides an interface to synchronize between spec parameters like Ports which have
// different constructs for different k8s types but are essentially the same.
// Change in Pod port => Change in Svc port but not vice versa.
type InternalWavefrontProxy struct {
	// The actual deep copy of instance
	instance *wfv1.WavefrontProxy

	// -----Wavefront Proxy CR Instance/Instance Status related.----- //
	// boolean value to indicate if the CR is newly created/updated during reconcile.
	// This is used to update the CR instance/status. Not used to identify auto upgrades.
	updateCR bool

	// -----Wavefront Proxy Deployment related.----- //
	// The container ports required to be opened up for wavefront proxy
	ContainerPorts    []corev1.ContainerPort
	ContainerPortsMap map[string]corev1.ContainerPort

	// The Volume Mounts of Container and Volumes Sources for Proxy
	volumeMount []corev1.VolumeMount
	volume      []corev1.Volume

	// The Environment args that need to be set for wavefront proxy
	EnvWavefrontProxyArgs string

	// ------Wavefront Proxy Service related.------ //
	// The container ports required to be opened up for wavefront proxy service.
	ServicePorts    []corev1.ServicePort
	ServicePortsMap map[string]corev1.ServicePort
}

func (ip *InternalWavefrontProxy) initialize(instance *wfv1.WavefrontProxy, reqLogger logr.Logger) {
	ip.instance = instance

	// Configs with defaults.
	// Proxy related
	// Proxy Image
	if ip.instance.Spec.Image == "" {
		ip.instance.Spec.Image = defaultImage
	}
	
	imgSlice := strings.Split(ip.instance.Spec.Image, ":")
	// Validate Image format and Auto Upgrade.
	if len(imgSlice) == 2 {
		ip.instance.Status.Version = imgSlice[1]
		if ip.instance.Spec.EnableAutoUpgrade {
			finalVer, err := util.GetLatestVersion(ip.instance.Spec.Image, reqLogger)
			if err == nil && finalVer != "" {
				ip.instance.Status.Version = finalVer
				ip.instance.Spec.Image = imgSlice[0] + ":" + finalVer
			} else if err != nil {
				reqLogger.Error(err, "Auto Upgrade Error.")
			}
		}
	} else {
		reqLogger.Info("Cannot update CR's Status.version", "Un-recognized format for CR Image", instance.Spec.Image)
	}

	ip.ContainerPortsMap = make(map[string]corev1.ContainerPort)
	ip.ServicePortsMap = make(map[string]corev1.ServicePort)

	// Below order shouldn't change, Any additional properties should only be appended at the bottom of this function.
	var envProxyArgs strings.Builder
	if ip.instance.Spec.MetricPort == 0 {
		ip.instance.Spec.MetricPort = defaultMetricPort
	}

	ip.addPort("metricport", ip.instance.Spec.MetricPort)
	envProxyArgs.WriteString(" --pushListenerPorts " + strconv.FormatInt(int64(ip.instance.Spec.MetricPort), 10))

	// Configs without any defaults.
	if ip.instance.Spec.TracePort != 0 {
		ip.addPort("traceport", ip.instance.Spec.TracePort)
		envProxyArgs.WriteString(" --traceListenerPorts " + strconv.FormatInt(int64(ip.instance.Spec.TracePort), 10))
	}

	if ip.instance.Spec.JaegerPort != 0 {
		ip.addPort("jaegerport", ip.instance.Spec.JaegerPort)
		envProxyArgs.WriteString(" --traceJaegerListenerPorts " + strconv.FormatInt(int64(ip.instance.Spec.JaegerPort), 10))
	}

	if ip.instance.Spec.ZipkinPort != 0 {
		ip.addPort("zipkinport", ip.instance.Spec.ZipkinPort)
		envProxyArgs.WriteString(" --traceZipkinListenerPorts " + strconv.FormatInt(int64(ip.instance.Spec.ZipkinPort), 10))
	}

	if ip.instance.Spec.HistogramDistPort != 0 {
		ip.addPort("histdistport", ip.instance.Spec.HistogramDistPort)
		envProxyArgs.WriteString(" --histogramDistListenerPorts " + strconv.FormatInt(int64(ip.instance.Spec.HistogramDistPort), 10))
	}

	// Configs with no defaults.
	if ip.instance.Spec.TraceSamplingRate != 0 {
		envProxyArgs.WriteString(" --traceSamplingRate " + strconv.FormatFloat(ip.instance.Spec.TraceSamplingRate, 'E', -1, 64))
	}

	if ip.instance.Spec.TraceSamplingDuration != 0 {
		envProxyArgs.WriteString(" --traceSamplingDuration " + strconv.FormatFloat(ip.instance.Spec.TraceSamplingDuration, 'E', -1, 64))
	}

	ip.volumeMount = make([]corev1.VolumeMount, 0, 2)
	ip.volume = make([]corev1.Volume, 0, 2)
	if ip.instance.Spec.Openshift {
		ip.volumeMount = append(ip.volumeMount, corev1.VolumeMount{
			Name:      "wavefront-proxy-storage",
			MountPath: "/var/spool/wavefront-proxy",
		})

		ip.volume = append(ip.volume, corev1.Volume{
			Name: "wavefront-proxy-storage",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: ip.instance.Spec.StorageClaimName,
				},
			},
		})
	}

	if ip.instance.Spec.Preprocessor != "" {
		envProxyArgs.WriteString(" --preprocessorConfigFile " + deafultPreprocessorMountPath + "/rules.yaml")
		ip.volumeMount = append(ip.volumeMount, corev1.VolumeMount{
			Name:      "preprocessor-volume",
			MountPath: deafultPreprocessorMountPath,
			ReadOnly:  true,
		})

		ip.volume = append(ip.volume, corev1.Volume{
			Name: "preprocessor-volume",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: ip.instance.Spec.Preprocessor},
				},
			},
		})
	}

	if ip.instance.Spec.Advanced != "" {
		envProxyArgs.WriteString(" -f " + defaultAdvancedWavefrontConfPath + "/wavefront.conf")
		ip.volumeMount = append(ip.volumeMount, corev1.VolumeMount{
			Name:      "advanced-volume",
			MountPath: defaultAdvancedWavefrontConfPath,
			ReadOnly:  true,
		})

		ip.volume = append(ip.volume, corev1.Volume{
			Name: "advanced-volume",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: ip.instance.Spec.Advanced},
				},
			},
		})

		// Add AdditionalPorts (if any) specified.
		// AdditonalPorts are meant for use with only Advanced config.
		ports := getCommaSeparatedPorts(ip.instance.Spec.AdditionalPorts)
		for i := range ports {
			if ports[i] != "" {
				if port, err := strconv.Atoi(ports[i]); err == nil {
					ip.addPort("port"+strconv.Itoa(i), int32(port))
				}
			}
		}
	}

	ip.EnvWavefrontProxyArgs = envProxyArgs.String()

	// Convert from map to slices.
	ip.ContainerPorts = make([]corev1.ContainerPort, 0, len(ip.ContainerPortsMap))
	for _, v := range ip.ContainerPortsMap {
		ip.ContainerPorts = append(ip.ContainerPorts, v)
	}

	ip.ServicePorts = make([]corev1.ServicePort, 0, len(ip.ServicePorts))
	for _, v := range ip.ServicePortsMap {
		ip.ServicePorts = append(ip.ServicePorts, v)
	}
}

func (ip *InternalWavefrontProxy) addPort(name string, port int32) {
	//Store in maps to maintain uniqueness of ports.
	ip.ContainerPortsMap[name] = corev1.ContainerPort{Name: name, ContainerPort: port}
	ip.ServicePortsMap[name] = corev1.ServicePort{Name: name, Port: port}
}
