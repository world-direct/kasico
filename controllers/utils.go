package controllers

import (
	"crypto/md5"
	"encoding/hex"
	"sort"

	kasicov1 "github.com/world-direct/kasico/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HashStringMap returns a string representing a Hash over the name
// and the keys and values within items. This is used to get the hashes of ConfigMap data
func HashStringMap(items map[string]string) string {
	md5 := md5.New()

	keys := make([]string, len(items))
	for key := range items {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		md5.Write([]byte(key))
		md5.Write([]byte(items[key]))
	}

	final := "md5:" + hex.EncodeToString(md5.Sum(nil))
	return final
}

func GetRoutingData(routerInstance kasicov1.RouterInstance, allIngresses []kasicov1.Ingress) *RoutingData {

	rd := &RoutingData{
		UDPPort:          routerInstance.Spec.RouterService.UDPPort,
		TCPPort:          routerInstance.Spec.RouterService.TCPPort,
		AdvertiseAddress: routerInstance.Spec.RouterService.AdvertiseAddress,
		Generation:       0,
	}

	rules := []RoutingRule{}
	for _, ingress := range allIngresses {

		if ingress.Spec.IngressClassName != routerInstance.Spec.IngressClassName {
			continue
		}

		owner := ingress.Namespace + "/" + ingress.Name
		for _, rule := range ingress.Spec.Rules {
			rules = append(rules, RoutingRule{
				Owner:      owner,
				Domain:     rule.Sip.Domain,
				Headnumber: rule.Sip.Headnumber,
				Backend:    rule.Backend.Service.Name + "." + ingress.Namespace,
			})
		}
	}

	rd.Rules = rules

	return rd

}

func SetLabel(metadata *metav1.ObjectMeta, key string, value string) {
	if metadata.Labels == nil {
		metadata.Labels = make(map[string]string)
	}

	metadata.Labels[key] = value
}

func GetLabel(metadata *metav1.ObjectMeta, key string) string {
	if metadata.Labels == nil {
		return ""
	}

	for lkey, lvalue := range metadata.Labels {
		if key == lkey {
			return lvalue
		}
	}

	return ""
}

func SetAnnotation(metadata *metav1.ObjectMeta, key string, value string) {
	if metadata.Annotations == nil {
		metadata.Annotations = make(map[string]string)
	}

	metadata.Annotations[key] = value
}

func GetAnnotation(metadata *metav1.ObjectMeta, key string) string {
	if metadata.Annotations == nil {
		return ""
	}

	for lkey, lvalue := range metadata.Annotations {
		if key == lkey {
			return lvalue
		}
	}

	return ""
}
