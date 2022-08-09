/*
Copyright The ORAS Authors.
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

package option

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/pflag"
	"oras.land/oras-go/v2/content"
)

const (
	annotationManifest = "$manifest"
)

var (
	errAnnotationConflict    = errors.New("annotations cannot be specified via flags and file at the same time")
	errAnnotationFormat      = errors.New("annotation MUST be a key-value pair")
	errAnnotationDuplication = errors.New("annotation key duplication")
)

// Packer option struct.
type Packer struct {
	ManifestExportPath     string
	PathValidationDisabled bool
	AnnotationsFilePath    string
	ManifestAnnotations    []string

	FileRefs []string
}

// ApplyFlags applies flags to a command flag set.
func (opts *Packer) ApplyFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&opts.ManifestExportPath, "export-manifest", "", "", "export the pushed manifest")
	fs.StringArrayVarP(&opts.ManifestAnnotations, "annotation", "a", nil, "manifest annotations")
	fs.StringVarP(&opts.AnnotationsFilePath, "annotations-file", "", "", "path of the annotation file")
	fs.BoolVarP(&opts.PathValidationDisabled, "disable-path-validation", "", false, "skip path validation")
}

// ExportManifest saves the pushed manifest to a local file.
func (opts *Packer) ExportManifest(ctx context.Context, fetcher content.Fetcher, desc ocispec.Descriptor) error {
	if opts.ManifestExportPath == "" {
		return nil
	}
	manifestBytes, err := content.FetchAll(ctx, fetcher, desc)
	if err != nil {
		return err
	}
	return os.WriteFile(opts.ManifestExportPath, manifestBytes, 0666)
}

// LoadManifestAnnotations loads the manifest annotation map.
func (opts *Packer) LoadManifestAnnotations() (annotations map[string]map[string]string, err error) {
	annotations = make(map[string]map[string]string)
	if opts.AnnotationsFilePath != "" && len(opts.ManifestAnnotations) != 0 {
		return nil, errAnnotationConflict
	}
	if opts.AnnotationsFilePath != "" {
		if err = decodeJSON(opts.AnnotationsFilePath, &annotations); err != nil {
			return nil, err
		}
	}
	if annotationsLength := len(opts.ManifestAnnotations); annotationsLength != 0 {
		if err = parseAnnotationFlags(opts.ManifestAnnotations, annotations); err != nil {
			return nil, err
		}
	}
	return annotations, nil
}

// decodeJSON decodes a json file v to filename.
func decodeJSON(filename string, v interface{}) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewDecoder(file).Decode(v)
}

// parseAnnotationFlags resharps annotationslice to k-v type and updates annotations
func parseAnnotationFlags(ManifestAnnotations []string, annotations map[string]map[string]string) error {
	rawAnnotationsMap := make(map[string]string)
	for _, anno := range ManifestAnnotations {
		parts := strings.SplitN(anno, "=", 2)
		if len(parts) != 2 {
			return errAnnotationFormat
		}
		parts[0], parts[1] = strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		if _, ok := rawAnnotationsMap[parts[0]]; ok {

			return fmt.Errorf("found annotation key, %v, more than once, %w", parts[0], errAnnotationDuplication)
		}
		rawAnnotationsMap[parts[0]] = parts[1]
	}
	annotations[annotationManifest] = rawAnnotationsMap
	return nil
}