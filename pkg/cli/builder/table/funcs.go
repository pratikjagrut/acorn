package table

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	adminv1 "github.com/acorn-io/runtime/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/tags"
	"github.com/rancher/wrangler/pkg/data/convert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var (
	FuncMap = map[string]any{
		"ago":           FormatCreated,
		"json":          FormatJSON,
		"jsoncompact":   FormatJSONCompact,
		"yaml":          FormatYAML,
		"first":         FormatFirst,
		"toJson":        ToJSON,
		"boolToStar":    BoolToStar,
		"array":         ToArray,
		"arrayFirst":    ToArrayFirst,
		"arrayNoSpace":  ToArrayNoSpace,
		"graph":         Graph,
		"pointer":       Pointer,
		"fullID":        FormatID,
		"name":          Name,
		"trunc":         Trunc,
		"alias":         Noop,
		"appGeneration": AppGeneration,
		"displayRange":  DisplayRange,
		"memoryToRange": MemoryToRange,
		"defaultMemory": DefaultMemory,
		"ownerName":     OwnerReferenceName,
		"imageName":     ImageName,
		"imageCommit":   ImageCommit,
	}
)

func Name(obj any) (string, error) {
	ro, ok := toKObject(obj)
	if ok {
		return ro.GetName(), nil
	}
	return "", fmt.Errorf("invalid obj %T", obj)
}

func Noop(obj any) string {
	return ""
}

func Trunc(s string) string {
	if tags.SHAPattern.MatchString(s) && len(s) > 12 {
		return s[:12]
	}
	if tags.CommitPattern.MatchString(s) && len(s) > 12 {
		return s[:12]
	}
	return s
}

func ToArray(s []string) (string, error) {
	return strings.Join(s, ", "), nil
}

func ToArrayNoSpace(s []string) (string, error) {
	return strings.Join(s, ","), nil
}

func ToArrayFirst(s []string) (string, error) {
	if len(s) > 0 {
		return s[0], nil
	}
	return "", nil
}

func Graph(value int) (string, error) {
	bars := int(float64(value) / 100.0 * 30)
	builder := &strings.Builder{}
	for i := 0; i < bars; i++ {
		if i == bars-1 {
			builder.WriteString(fmt.Sprintf("> %v", value))
			break
		}
		builder.WriteString("=")
	}
	return builder.String(), nil
}

func Pointer(data any) string {
	if reflect.ValueOf(data).IsNil() {
		return ""
	}
	return fmt.Sprint(data)
}

func FormatID(obj kclient.Object) (string, error) {
	return obj.GetName(), nil
}

func FormatCreated(data metav1.Time) string {
	return duration.HumanDuration(time.Now().UTC().Sub(data.Time)) + " ago"
}

func FormatJSON(data any) (string, error) {
	bytes, err := json.MarshalIndent(cleanFields(data), "", "    ")
	return string(bytes) + "\n", err
}

func FormatJSONCompact(data any) (string, error) {
	bytes, err := json.Marshal(cleanFields(data))
	return string(bytes) + "\n", err
}

func toKObject(obj any) (kclient.Object, bool) {
	ro, ok := obj.(kclient.Object)
	if !ok {
		newObj := reflect.New(reflect.TypeOf(obj))
		newObj.Elem().Set(reflect.ValueOf(obj))
		ro, ok = newObj.Interface().(kclient.Object)
	}
	return ro, ok
}

func cleanFields(obj any) any {
	if ol, ok := obj.(objectList); ok {
		for i, o := range ol.Items {
			ol.Items[i] = cleanFields(o)
		}
		return ol
	}

	ro, ok := toKObject(obj)
	if ok {
		ro.SetManagedFields(nil)
		ro.SetUID("")
		ro.SetGenerateName("")
		ro.SetResourceVersion("")
		labels := ro.GetLabels()
		for k := range labels {
			if strings.Contains(k, "acorn.io/") {
				delete(labels, k)
			}
		}
		ro.SetLabels(labels)

		annotations := ro.GetAnnotations()
		for k := range annotations {
			if strings.Contains(k, "acorn.io/") {
				delete(annotations, k)
			}
		}
		ro.SetAnnotations(annotations)

		// decode secret values
		if sec, ok := ro.(*apiv1.Secret); ok {
			return decodeSecret(sec)
		}

		return ro
	}
	return obj
}

func decodeSecret(sec *apiv1.Secret) any {
	decodedSecret := struct {
		metav1.TypeMeta   `json:",inline"`
		metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

		Type string            `json:"type,omitempty"`
		Data map[string]string `json:"data,omitempty"`
		Keys []string          `json:"keys,omitempty"`
	}{
		TypeMeta:   sec.TypeMeta,
		ObjectMeta: sec.ObjectMeta,
		Type:       sec.Type,
		Data:       map[string]string{},
		Keys:       sec.Keys,
	}
	for k, v := range sec.Data {
		decodedSecret.Data[k] = string(v)
	}
	return decodedSecret
}

func FormatYAML(data any) (string, error) {
	bytes, err := yaml.Marshal(cleanFields(data))
	return string(bytes) + "\n", err
}

func FormatFirst(data, data2 any) (string, error) {
	str := convert.ToString(data)
	if str != "" {
		return str, nil
	}

	str = convert.ToString(data2)
	if str != "" {
		return str, nil
	}

	return "", nil
}

func ToJSON(data any) (map[string]any, error) {
	return convert.EncodeToMap(data)
}

func BoolToStar(obj any) (string, error) {
	if b, ok := obj.(bool); ok && b {
		return "*", nil
	}
	if b, ok := obj.(*bool); ok && b != nil && *b {
		return "*", nil
	}
	return "", nil
}

func DisplayRange(minVal, maxVal any) (string, error) {
	min, max := fmt.Sprintf("%v", minVal), fmt.Sprintf("%v", maxVal)
	if max == "" {
		max = "Unrestricted"
	}
	if min == "" {
		if max == "Unrestricted" {
			return max, nil
		}
		min = "5M"
	}

	return fmt.Sprintf("%s-%s", min, max), nil
}

func DefaultMemory(obj any) (string, error) {
	b, ok := obj.(adminv1.ComputeClassMemory)
	if !ok {
		return "", fmt.Errorf("object passed is not a ComputeClassMemory struct")
	}

	result := b.Default
	if b.Default == "0" || b.Default == "" {
		result = b.Max
		if result == "0" || result == "" {
			result = "Unrestricted"
		}
	}
	return result, nil
}

func MemoryToRange(obj any) (string, error) {
	b, ok := obj.(adminv1.ComputeClassMemory)
	if !ok {
		return "", fmt.Errorf("object passed is not a ComputeClassMemory struct")
	}

	min := b.Min
	if min == "" {
		min = "0"
	}

	max := b.Max
	if max == "" || max == "0" {
		max = "Unrestricted"
	}

	if len(b.Values) != 0 {
		return strings.Join(b.Values, ","), nil
	}

	if max == "Unrestricted" && min == "0" {
		return "Unrestricted", nil
	}

	return fmt.Sprintf("%v-%v", min, max), nil
}

func AppGeneration(app apiv1.App, msg string) string {
	if app.Generation != app.Status.ObservedGeneration {
		return "[controller: not processed] " + msg
	}
	return msg
}

func OwnerReferenceName(obj metav1.Object) string {
	owners := obj.GetOwnerReferences()
	if len(owners) == 0 {
		return ""
	}

	return owners[0].Name
}

func ImageName(obj metav1.Object) string {
	app, ok := obj.(*apiv1.App)
	if !ok {
		return ""
	}

	suffix := ""
	if app.Status.ObservedAutoUpgrade {
		suffix = "*"
	}

	if original, exists := app.ObjectMeta.Annotations[labels.AcornOriginalImage]; exists {
		return original + suffix
	}
	return app.Status.AppImage.Name + suffix
}

func ImageCommit(obj metav1.Object) string {
	app, ok := obj.(*apiv1.App)
	if !ok {
		return ""
	}

	return app.Status.AppImage.VCS.Revision
}
