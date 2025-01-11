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

// Package stiface provides a set of interfaces for the types in
// cloud.google.com/go/storage. These can be used to create mocks or other test
// doubles. The package also provides adapters to enable the types of the
// storage package to implement these interfaces.
//
// We do not recommend using mocks for most testing. Please read
// https://testing.googleblog.com/2013/05/testing-on-toilet-dont-overuse-mocks.html.
//
// Note: This package is in alpha. Some backwards-incompatible changes may occur.
//
// You must embed these interfaces to implement them:
//
//    type ClientMock struct {
//        stiface.Client
//        ...
//    }
//
// This ensures that your implementations will not break when methods are added
// to the interfaces.
package stiface
