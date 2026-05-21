package s3

import (
	"context"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestCheckTags(t *testing.T) {
	tests := []struct {
		name    string
		tagging string
		wantErr string
	}{
		{name: "empty", tagging: ""},
		{name: "single tag", tagging: "key=value"},
		{name: "multiple tags", tagging: "k1=v1&k2=v2&k3=v3"},
		{name: "empty value allowed", tagging: "key="},
		{name: "equals in value", tagging: "k=a=b"},
		{name: "max key length", tagging: strings.Repeat("a", 128) + "=v"},
		{name: "max value length", tagging: "k=" + strings.Repeat("a", 256)},
		{name: "ten tags", tagging: strings.Repeat("k=v&", 9) + "k=v"},

		{
			name:    "missing equals",
			tagging: "keyvalue",
			wantErr: `invalid tag "keyvalue": expected key=value`,
		},
		{
			name:    "empty key",
			tagging: "=value",
			wantErr: `invalid tag "=value": expected key=value`,
		},
		{
			name:    "too many tags",
			tagging: strings.Repeat("k=v&", 10) + "k=v",
			wantErr: "S3 allows at most 10 tags per object",
		},
		{
			name:    "key too long",
			tagging: strings.Repeat("a", 129) + "=v",
			wantErr: "exceeds 128 unicode characters",
		},
		{
			name:    "value too long",
			tagging: "k=" + strings.Repeat("a", 257),
			wantErr: "exceeds 256 unicode characters",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := checkTags(tc.tagging)
			if tc.wantErr == "" {
				assert.NilError(t, err)
				return
			}
			assert.ErrorContains(t, err, tc.wantErr)
		})
	}
}

// TestObjectStore_newPutObjectInput_TaggingHeader pins the fix for
// loft-sh/vcluster#3882: with no tagging configured, the resulting
// PutObjectInput must leave Tagging nil so the AWS SDK does not emit
// an empty x-amz-tagging header that Cloudflare R2 and other
// S3-compatible backends reject with HTTP 501.
func TestObjectStore_newPutObjectInput_TaggingHeader(t *testing.T) {
	tests := []struct {
		name           string
		tagging        string
		wantTaggingSet bool
		wantTagging    string
	}{
		{name: "empty tagging leaves header unset", tagging: ""},
		{
			name:           "configured tagging sets header",
			tagging:        "env=prod&team=core",
			wantTaggingSet: true,
			wantTagging:    "env=prod&team=core",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o := &ObjectStore{bucket: "b", key: "k", tagging: tc.tagging}

			input := o.newPutObjectInput(context.Background(), strings.NewReader(""))

			if !tc.wantTaggingSet {
				assert.Assert(t, input.Tagging == nil)
				return
			}
			assert.Assert(t, input.Tagging != nil)
			assert.Equal(t, *input.Tagging, tc.wantTagging)
		})
	}
}
