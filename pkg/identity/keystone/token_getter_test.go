/*
Copyright 2018 The Kubernetes Authors.
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

package keystone

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestTokenGetter(t *testing.T) {
	th.SetupHTTP()
	defer th.TeardownHTTP()

	const ID = "0123456789"

	th.Mux.HandleFunc("/v3/auth/tokens", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("X-Subject-Token", ID)
		type AuthRequest struct {
			Auth struct {
				Identity struct {
					Password struct {
						User struct {
							Domain   struct{ Name string }
							Name     string
							Password string
						}
					}
				}
			}
		}
		var x AuthRequest
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &x)
		domainName := x.Auth.Identity.Password.User.Domain.Name
		userName := x.Auth.Identity.Password.User.Name
		password := x.Auth.Identity.Password.User.Password
		if domainName == "default" && userName == "testuser" && password == "testpw" {
			w.WriteHeader(http.StatusCreated)
			resp := `{"token": {
							"methods": [
								"password"
							],
							"expires_at": "2015-11-09T01:42:57.527363Z",
							"user": {
								"domain": {
									"id": "default",
									"name": "Default"
								},
								"id": "some_id",
								"name": "admin",
								"password_expires_at": null
							},
							"audit_ids": [
								"lC2Wj1jbQe-dLjLyOx4qPQ"
							],
							"issued_at": "2015-11-09T00:42:57.527404Z"
						}
					}`
			fmt.Fprint(w, resp)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	})

	// Correct password
	options := Options{
		AuthOptions: gophercloud.AuthOptions{
			IdentityEndpoint: th.Endpoint(),
			Username:         "testuser",
			Password:         "testpw",
			DomainName:       "default",
		},
	}

	token, err := GetToken(context.TODO(), options)
	th.AssertNoErr(t, err)
	th.AssertEquals(t, "0123456789", token.ID)
	th.AssertEquals(t, "2015-11-09 01:42:57.527363 +0000 UTC", token.ExpiresAt.String())

	// Incorrect password
	options.AuthOptions.Password = "wrongpw"

	_, err = GetToken(context.TODO(), options)
	if !gophercloud.ResponseCodeIs(err, http.StatusUnauthorized) {
		t.FailNow()
	}

	// Invalid auth data
	options.AuthOptions.Password = ""

	_, err = GetToken(context.TODO(), options)
	th.AssertEquals(t, "You must provide a password to authenticate", err.Error())
}
