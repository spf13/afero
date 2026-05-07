// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stiface_test

import (
	"context"

	"cloud.google.com/go/storage"
	"github.com/spf13/afero/gcsfs/internal/stiface"
)

func Example_AdaptClient() {
	ctx := context.Background()
	c, err := storage.NewClient(ctx)
	if err != nil {
		// TODO: Handle error.
	}
	client := stiface.AdaptClient(c)
	w := client.Bucket("my-bucket").Object("my-object").NewWriter(ctx)
	w.ObjectAttrs().ContentType = "text/plain"
	// TODO: Use w.
}
