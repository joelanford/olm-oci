package cli

import (
	"context"
	"fmt"
	"log"
	"os"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"sigs.k8s.io/yaml"

	pkg "github.com/joelanford/olm-oci/api/v1"
	"github.com/joelanford/olm-oci/pkg/inspect"
	"github.com/joelanford/olm-oci/pkg/remote"
)

func NewAnnotateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "annotate",
		Short: "Annotate an OLM OCI artifact",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			subjectRef := args[0]
			annotationsFile := args[1]

			if err := runAnnotate(cmd.Context(), subjectRef, annotationsFile); err != nil {
				log.Fatal(err)
			}
		},
	}
	return cmd
}

func runAnnotate(ctx context.Context, subjectRef, annotationsFile string) error {
	annotationsData, err := os.ReadFile(annotationsFile)
	if err != nil {
		return fmt.Errorf("read annotations file: %v", err)
	}

	var annotations map[string]string
	if err := yaml.Unmarshal(annotationsData, &annotations); err != nil {
		return fmt.Errorf("parse annotations file: %v", err)
	}

	subjectRepo, _, subjectDesc, err := remote.ResolveNameAndReference(ctx, subjectRef)
	if err != nil {
		return fmt.Errorf("parse subject reference: %v", err)
	}

	if subjectDesc.MediaType != ocispec.MediaTypeArtifactManifest {
		return fmt.Errorf("subject reference must be an artifact manifest")
	}

	artifactReader, err := subjectRepo.Fetch(ctx, *subjectDesc)
	if err != nil {
		return fmt.Errorf("fetch artifact manifest: %v", err)
	}
	defer artifactReader.Close()
	art, err := inspect.DecodeArtifact(artifactReader)
	if err != nil {
		return fmt.Errorf("decode artifact manifest: %v", err)
	}
	switch art.ArtifactType {
	case pkg.MediaTypeCatalog, pkg.MediaTypePackage, pkg.MediaTypeChannel, pkg.MediaTypeBundle:
	default:
		return fmt.Errorf("subject reference must be a catalog, package, channel, or bundle")
	}

	annotationsDesc, err := oras.Pack(ctx, subjectRepo, pkg.MediaTypeAnnotations, nil, oras.PackOptions{
		Subject:             subjectDesc,
		ManifestAnnotations: annotations,
	})
	if err != nil {
		return fmt.Errorf("pack annotations: %v", err)
	}
	fmt.Printf("Sucessfully annotated artifact manifest %q with annotations at digest %s\n", subjectRef, annotationsDesc.Digest)
	return nil
}
