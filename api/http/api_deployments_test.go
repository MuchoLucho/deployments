// Copyright 2021 Northern.tech AS
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package http

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mendersoftware/deployments/app"
	mapp "github.com/mendersoftware/deployments/app/mocks"
	"github.com/mendersoftware/deployments/model"
	"github.com/mendersoftware/deployments/utils/restutil/view"
	h "github.com/mendersoftware/deployments/utils/testing"
	"github.com/mendersoftware/go-lib-micro/identity"
	"github.com/mendersoftware/go-lib-micro/requestid"
	"github.com/mendersoftware/go-lib-micro/rest_utils"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
)

func TestAlive(t *testing.T) {
	t.Parallel()

	req, _ := http.NewRequest("GET", "http://localhost"+ApiUrlInternalAlive, nil)
	d := NewDeploymentsApiHandlers(nil, nil, nil)
	api := setUpRestTest(ApiUrlInternalAlive, rest.Get, d.AliveHandler)
	recorded := test.RunRequest(t, api.MakeHandler(), req)
	recorded.CodeIs(http.StatusNoContent)
}

func TestHealthCheck(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		Name string

		AppError     error
		ResponseCode int
		ResponseBody interface{}
	}{{
		Name:         "ok",
		ResponseCode: http.StatusNoContent,
	}, {
		Name:         "error: app unhealthy",
		AppError:     errors.New("*COUGH! COUGH!*"),
		ResponseCode: http.StatusServiceUnavailable,
		ResponseBody: rest_utils.ApiError{
			Err:   "*COUGH! COUGH!*",
			ReqId: "test",
		},
	}}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			app := &mapp.App{}
			app.On("HealthCheck", mock.MatchedBy(
				func(ctx interface{}) bool {
					if _, ok := ctx.(context.Context); ok {
						return true
					}
					return false
				}),
			).Return(tc.AppError)
			d := NewDeploymentsApiHandlers(nil, nil, app)
			api := setUpRestTest(
				ApiUrlInternalHealth,
				rest.Get,
				d.HealthHandler,
			)
			req, _ := http.NewRequest(
				"GET",
				"http://localhost"+ApiUrlInternalHealth,
				nil,
			)
			req.Header.Set("X-MEN-RequestID", "test")
			recorded := test.RunRequest(t, api.MakeHandler(), req)
			recorded.CodeIs(tc.ResponseCode)
			if tc.ResponseBody != nil {
				b, _ := json.Marshal(tc.ResponseBody)
				assert.JSONEq(t, string(b), recorded.Recorder.Body.String())
			} else {
				recorded.BodyIs("")
			}
		})
	}
}

func TestDeploymentsPerTenantHandler(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		tenant       string
		queryString  string
		appError     error
		query        *model.Query
		deployments  []*model.Deployment
		count        int64
		responseCode int
		responseBody interface{}
	}{
		"ok": {
			tenant: "tenantID",
			query: &model.Query{
				Limit: rest_utils.PerPageDefault + 1,
			},
			deployments:  []*model.Deployment{},
			count:        0,
			responseCode: http.StatusOK,
			responseBody: []*model.Deployment{},
		},
		"ok with pagination": {
			tenant:      "tenantID",
			queryString: rest_utils.PerPageName + "=50&" + rest_utils.PageName + "=2",
			query: &model.Query{
				Skip:  50,
				Limit: 51,
			},
			deployments:  []*model.Deployment{},
			count:        0,
			responseCode: http.StatusOK,
			responseBody: []*model.Deployment{},
		},
		"ko, missing tenant ID": {
			tenant:       "",
			responseCode: http.StatusBadRequest,
			responseBody: rest_utils.ApiError{
				Err:   "missing tenant ID",
				ReqId: "test",
			},
		},
		"ko, error in pagination": {
			tenant:       "tenantID",
			queryString:  rest_utils.PerPageName + "=a",
			responseCode: http.StatusBadRequest,
			responseBody: rest_utils.ApiError{
				Err:   "Can't parse param per_page",
				ReqId: "test",
			},
		},
		"ko, error in filters": {
			tenant:       "tenantID",
			queryString:  "created_before=a",
			responseCode: http.StatusBadRequest,
			responseBody: rest_utils.ApiError{
				Err:   "timestamp parsing failed for created_before parameter: invalid timestamp: a",
				ReqId: "test",
			},
		},
		"ko, error in LookupDeployment": {
			tenant: "tenantID",
			query: &model.Query{
				Limit: rest_utils.PerPageDefault + 1,
			},
			appError:     errors.New("generic error"),
			deployments:  []*model.Deployment{},
			count:        0,
			responseCode: http.StatusBadRequest,
			responseBody: rest_utils.ApiError{
				Err:   "generic error",
				ReqId: "test",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			app := &mapp.App{}
			if tc.query != nil {
				app.On("LookupDeployment",
					mock.MatchedBy(func(ctx context.Context) bool {
						return true
					}),
					*tc.query,
				).Return(tc.deployments, tc.count, tc.appError)
			}
			defer app.AssertExpectations(t)

			restView := new(view.RESTView)
			d := NewDeploymentsApiHandlers(nil, restView, app)
			api := setUpRestTest(
				ApiUrlInternalTenantDeployments,
				rest.Get,
				d.DeploymentsPerTenantHandler,
			)

			url := strings.Replace(ApiUrlInternalTenantDeployments, ":tenant", tc.tenant, 1)
			if tc.queryString != "" {
				url = url + "?" + tc.queryString
			}
			req, _ := http.NewRequest(
				"GET",
				"http://localhost"+url,
				bytes.NewReader([]byte("")),
			)
			req.Header.Set("X-MEN-RequestID", "test")
			recorded := test.RunRequest(t, api.MakeHandler(), req)
			recorded.CodeIs(tc.responseCode)
			if tc.responseBody != nil {
				b, _ := json.Marshal(tc.responseBody)
				assert.JSONEq(t, string(b), recorded.Recorder.Body.String())
			} else {
				recorded.BodyIs("")
			}
		})
	}
}

func TestPostDeployment(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		Name      string
		InputBody interface{}

		AppError               error
		ResponseCode           int
		ResponseLocationHeader string
		ResponseBody           interface{}
	}{{
		Name: "ok, device list",
		InputBody: &model.DeploymentConstructor{
			Name:         "foo",
			ArtifactName: "bar",
			Devices:      []string{"f826484e-1157-4109-af21-304e6d711560"},
		},
		ResponseCode:           http.StatusCreated,
		ResponseLocationHeader: "./management/v1/deployments/deployments/foo",
	}, {
		Name: "ok, all devices",
		InputBody: &model.DeploymentConstructor{
			Name:         "foo",
			ArtifactName: "bar",
			AllDevices:   true,
		},
		ResponseCode:           http.StatusCreated,
		ResponseLocationHeader: "./management/v1/deployments/deployments/foo",
	}, {
		Name:         "error: empty payload",
		ResponseCode: http.StatusBadRequest,
		ResponseBody: rest_utils.ApiError{
			Err:   "Validating request body: JSON payload is empty",
			ReqId: "test",
		},
	}, {
		Name: "error: app error",
		InputBody: &model.DeploymentConstructor{
			Name:         "foo",
			ArtifactName: "bar",
			AllDevices:   true,
		},
		AppError:     errors.New("some error"),
		ResponseCode: http.StatusInternalServerError,
		ResponseBody: rest_utils.ApiError{
			Err:   "internal error",
			ReqId: "test",
		},
	}, {
		Name: "error: app error: no devices",
		InputBody: &model.DeploymentConstructor{
			Name:         "foo",
			ArtifactName: "bar",
			AllDevices:   true,
		},
		AppError:     app.ErrNoDevices,
		ResponseCode: http.StatusBadRequest,
		ResponseBody: rest_utils.ApiError{
			Err:   app.ErrNoDevices.Error(),
			ReqId: "test",
		},
	}, {
		Name: "error: conflict",
		InputBody: &model.DeploymentConstructor{
			Name:         "foo",
			ArtifactName: "bar",
			Devices:      []string{"f826484e-1157-4109-af21-304e6d711560"},
			AllDevices:   true,
		},
		ResponseCode: http.StatusBadRequest,
		ResponseBody: rest_utils.ApiError{
			Err:   "Validating request body: Invalid deployments definition: list of devices provided togheter with all_devices flag",
			ReqId: "test",
		},
	}, {
		Name: "error: no devices",
		InputBody: &model.DeploymentConstructor{
			Name:         "foo",
			ArtifactName: "bar",
		},
		ResponseCode: http.StatusBadRequest,
		ResponseBody: rest_utils.ApiError{
			Err:   "Validating request body: Invalid deployments definition: provide list of devices or set all_devices flag",
			ReqId: "test",
		},
	}}
	var constructor *model.DeploymentConstructor
	for _, tc := range testCases {
		if tc.InputBody != nil {
			constructor = tc.InputBody.(*model.DeploymentConstructor)
		} else {
			constructor = nil
		}
		t.Run(tc.Name, func(t *testing.T) {
			app := &mapp.App{}
			app.On("CreateDeployment", mock.MatchedBy(
				func(ctx interface{}) bool {
					if _, ok := ctx.(context.Context); ok {
						return true
					}
					return false
				}),
				constructor,
			).Return("foo", tc.AppError)
			restView := new(view.RESTView)
			d := NewDeploymentsApiHandlers(nil, restView, app)
			api := setUpRestTest(
				ApiUrlManagementDeployments,
				rest.Post,
				d.PostDeployment,
			)

			req := test.MakeSimpleRequest(
				"POST",
				"http://localhost"+ApiUrlManagementDeployments,
				tc.InputBody,
			)
			req.Header.Set("X-MEN-RequestID", "test")
			recorded := test.RunRequest(t, api.MakeHandler(), req)
			recorded.CodeIs(tc.ResponseCode)
			if tc.ResponseLocationHeader != "" {
				recorded.HeaderIs("Location", tc.ResponseLocationHeader)
			}
			if tc.ResponseBody != nil {
				b, _ := json.Marshal(tc.ResponseBody)
				assert.JSONEq(t, string(b), recorded.Recorder.Body.String())
			} else {
				recorded.BodyIs("")
			}
		})
	}
}

func TestPostDeploymentToGroup(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		Name       string
		InputBody  interface{}
		InputGroup string

		AppError               error
		ResponseCode           int
		ResponseLocationHeader string
		ResponseBody           interface{}
	}{{
		Name: "ok",
		InputBody: &model.DeploymentConstructor{
			Name:         "foo",
			ArtifactName: "bar",
		},
		InputGroup:             "baz",
		ResponseCode:           http.StatusCreated,
		ResponseLocationHeader: "./management/v1/deployments/deployments/foo",
	}, {
		Name:         "error: empty payload",
		InputGroup:   "baz",
		ResponseCode: http.StatusBadRequest,
		ResponseBody: rest_utils.ApiError{
			Err:   "Validating request body: JSON payload is empty",
			ReqId: "test",
		},
	}, {
		Name: "error: conflict",
		InputBody: &model.DeploymentConstructor{
			Name:         "foo",
			ArtifactName: "bar",
			Devices:      []string{"f826484e-1157-4109-af21-304e6d711560"},
			AllDevices:   true,
		},
		InputGroup:   "baz",
		ResponseCode: http.StatusBadRequest,
		ResponseBody: rest_utils.ApiError{
			Err:   "Validating request body: The deployment for group constructor should have neither list of devices nor all_devices flag set",
			ReqId: "test",
		},
	}, {
		Name: "error: app error",
		InputBody: &model.DeploymentConstructor{
			Name:         "foo",
			ArtifactName: "bar",
		},
		InputGroup:   "baz",
		AppError:     errors.New("some error"),
		ResponseCode: http.StatusInternalServerError,
		ResponseBody: rest_utils.ApiError{
			Err:   "internal error",
			ReqId: "test",
		},
	}, {
		Name: "error: app error: no devices",
		InputBody: &model.DeploymentConstructor{
			Name:         "foo",
			ArtifactName: "bar",
		},
		InputGroup:   "baz",
		AppError:     app.ErrNoDevices,
		ResponseCode: http.StatusBadRequest,
		ResponseBody: rest_utils.ApiError{
			Err:   app.ErrNoDevices.Error(),
			ReqId: "test",
		},
	}}
	var constructor *model.DeploymentConstructor
	for _, tc := range testCases {
		if tc.InputBody != nil {
			constructor = tc.InputBody.(*model.DeploymentConstructor)
			constructor.Group = tc.InputGroup
		} else {
			constructor = nil
		}
		t.Run(tc.Name, func(t *testing.T) {
			app := &mapp.App{}
			app.On("CreateDeployment", mock.MatchedBy(
				func(ctx interface{}) bool {
					if _, ok := ctx.(context.Context); ok {
						return true
					}
					return false
				}),
				constructor,
			).Return("foo", tc.AppError)
			restView := new(view.RESTView)
			d := NewDeploymentsApiHandlers(nil, restView, app)
			api := setUpRestTest(
				ApiUrlManagementDeploymentsGroup,
				rest.Post,
				d.DeployToGroup,
			)

			req := test.MakeSimpleRequest(
				"POST",
				"http://localhost"+ApiUrlManagementDeployments+"/group/"+tc.InputGroup,
				tc.InputBody,
			)
			req.Header.Set("X-MEN-RequestID", "test")
			recorded := test.RunRequest(t, api.MakeHandler(), req)
			recorded.CodeIs(tc.ResponseCode)
			if tc.ResponseLocationHeader != "" {
				recorded.HeaderIs("Location", tc.ResponseLocationHeader)
			}
			if tc.ResponseBody != nil {
				b, _ := json.Marshal(tc.ResponseBody)
				assert.JSONEq(t, string(b), recorded.Recorder.Body.String())
			} else {
				recorded.BodyIs("")
			}
		})
	}
}

func TestControllerPostConfigurationDeployment(t *testing.T) {

	t.Parallel()

	testCases := map[string]struct {
		h.JSONResponseParams

		InputBodyObject interface{}

		InputTenantID                           string
		InputDeviceID                           string
		InputDeploymentID                       string
		InputCreateConfigurationDeploymentError error
	}{
		"ok": {
			InputBodyObject: &model.ConfigurationDeploymentConstructor{
				Name:          "name",
				Configuration: "configuration",
			},
			InputTenantID:     "foo",
			InputDeviceID:     "bar",
			InputDeploymentID: "baz",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusCreated,
				OutputBodyObject: nil,
				OutputHeaders:    map[string]string{"Location": "./deployments/baz"},
			},
		},
		"ko, empty body": {
			InputBodyObject:   nil,
			InputTenantID:     "foo",
			InputDeviceID:     "bar",
			InputDeploymentID: "baz",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("Validating request body: JSON payload is empty")),
			},
		},
		"ko, empty deployment": {
			InputBodyObject:   &model.ConfigurationDeploymentConstructor{},
			InputTenantID:     "foo",
			InputDeviceID:     "bar",
			InputDeploymentID: "baz",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("Validating request body: configuration: cannot be blank; name: cannot be blank.")),
			},
		},
		"ko, internal error": {
			InputBodyObject: &model.ConfigurationDeploymentConstructor{
				Name:          "foo",
				Configuration: "bar",
			},
			InputTenantID:                           "foo",
			InputDeviceID:                           "bar",
			InputDeploymentID:                       "baz",
			InputCreateConfigurationDeploymentError: errors.New("model error"),
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusInternalServerError,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("internal error")),
			},
		},
		"ko, conflict": {
			InputBodyObject: &model.ConfigurationDeploymentConstructor{
				Name:          "foo",
				Configuration: "bar",
			},
			InputTenantID:                           "foo",
			InputDeviceID:                           "bar",
			InputDeploymentID:                       "baz",
			InputCreateConfigurationDeploymentError: app.ErrDuplicateDeployment,
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusConflict,
				OutputBodyObject: h.ErrorToErrStruct(app.ErrDuplicateDeployment),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(fmt.Sprintf("test case: %s", name), func(t *testing.T) {
			restView := new(view.RESTView)
			app := &mapp.App{}

			d := NewDeploymentsApiHandlers(nil, restView, app)

			app.On("CreateDeviceConfigurationDeployment",
				h.ContextMatcher(), mock.AnythingOfType("*model.ConfigurationDeploymentConstructor"),
				tc.InputDeviceID, tc.InputDeploymentID).
				Return(tc.InputDeploymentID, tc.InputCreateConfigurationDeploymentError)

			api := setUpRestTest(
				ApiUrlInternalDeviceConfigurationDeployments,
				rest.Post,
				d.PostDeviceConfigurationDeployment,
			)

			uri := strings.Replace(ApiUrlInternalDeviceConfigurationDeployments, ":tenant", tc.InputTenantID, 1)
			uri = strings.Replace(uri, ":device_id", tc.InputDeviceID, 1)
			uri = strings.Replace(uri, ":deployment_id", tc.InputDeploymentID, 1)

			req := test.MakeSimpleRequest("POST", "http://localhost"+uri, tc.InputBodyObject)
			req.Header.Add(requestid.RequestIdHeader, "test")

			recorded := test.RunRequest(t, api.MakeHandler(), req)

			h.CheckRecordedResponse(t, recorded, tc.JSONResponseParams)
		})
	}
}

type brokenReader struct{}

func (r brokenReader) Read(b []byte) (int, error) {
	return 0, errors.New("rekt")
}

func TestDownloadConfiguration(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name string

		Config  *Config
		Request *http.Request
		App     *mapp.App // mock App

		// Response parameters
		StatusCode int    // Response StatusCode
		Error      error  // Error message in case of non-2XX response.
		Body       []byte // The Body on 2XX responses.
		Headers    http.Header
	}{{
		Name: "ok",

		Request: func() *http.Request {
			req, _ := http.NewRequest(
				http.MethodGet,
				FMTConfigURL(
					"http", "localhost",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")).String(),
					"Bagelbone",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				),
				nil,
			)
			sig := model.NewRequestSignature(req, []byte("test"))
			sig.SetExpire(time.Now().Add(time.Minute))
			signature := sig.HMAC256()
			q := req.URL.Query()
			q.Set(
				model.ParamSignature,
				base64.RawURLEncoding.EncodeToString(signature))
			req.URL.RawQuery = q.Encode()
			return req
		}(),
		Config: NewConfig().
			SetPresignExpire(time.Minute).
			SetPresignSecret([]byte("test")).
			SetPresignHostname("localhost").
			SetPresignScheme("http"),
		App: func() *mapp.App {
			app := new(mapp.App)
			app.On("GenerateConfigurationImage",
				contextMatcher(),
				"Bagelbone",
				uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")).String(),
			).Return(bytes.NewReader([]byte("*Just imagine an artifact here*")), nil)
			return app
		}(),

		Headers: http.Header{
			"Content-Disposition": []string{"attachment; filename=\"artifact.mender\""},
			"Content-Type":        []string{app.ArtifactContentType},
		},
		StatusCode: http.StatusOK,
		Body:       []byte("*Just imagine an artifact here*"),
	}, {
		Name: "ok, multi-tenant",

		Request: func() *http.Request {
			req, _ := http.NewRequest(
				http.MethodGet,
				FMTConfigURL(
					"http", "localhost",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")).String(),
					"Bagelbone",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				),
				nil,
			)
			sig := model.NewRequestSignature(req, []byte("test"))
			sig.SetExpire(time.Now().Add(time.Minute))
			q := req.URL.Query()
			q.Set("tenant_id", "123456789012345678901234")
			req.URL.RawQuery = q.Encode()
			signature := sig.HMAC256()
			q.Set(
				model.ParamSignature,
				base64.RawURLEncoding.EncodeToString(signature))
			req.URL.RawQuery = q.Encode()
			return req
		}(),
		Config: NewConfig().
			SetPresignExpire(time.Minute).
			SetPresignSecret([]byte("test")).
			SetPresignHostname("localhost").
			SetPresignScheme("http"),
		App: func() *mapp.App {
			app := new(mapp.App)
			app.On("GenerateConfigurationImage",
				contextMatcher(),
				"Bagelbone",
				uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")).String(),
			).Return(bytes.NewReader([]byte("*Just imagine an artifact here*")), nil)
			return app
		}(),

		Headers: http.Header{
			"Content-Disposition": []string{"attachment; filename=\"artifact.mender\""},
			"Content-Type":        []string{app.ArtifactContentType},
		},
		StatusCode: http.StatusOK,
		Body:       []byte("*Just imagine an artifact here*"),
	}, {
		Name: "error, signing configured incorrectly",

		Request: func() *http.Request {
			req, _ := http.NewRequest(
				http.MethodGet,
				FMTConfigURL(
					"http", "localhost",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")).String(),
					"Bagelbone",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				),
				nil,
			)
			return req
		}(),
		App: new(mapp.App),

		StatusCode: http.StatusNotFound,
		Error:      errors.New("Resource not found"),
	}, {
		Name: "error, invalid request",

		Config: NewConfig().
			SetPresignSecret([]byte("test")),
		Request: func() *http.Request {
			req, _ := http.NewRequest(
				http.MethodGet,
				FMTConfigURL(
					"http", "localhost",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")).String(),
					"Bagelbone",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				),
				nil,
			)
			return req
		}(),
		App: new(mapp.App),

		StatusCode: http.StatusBadRequest,
		Error: errors.New("invalid request parameters: " +
			"x-men-expire: required key is missing; " +
			"x-men-signature: required key is missing.",
		),
	}, {
		Name: "error, signature expired",

		Config: NewConfig().
			SetPresignSecret([]byte("test")),
		Request: func() *http.Request {
			req, _ := http.NewRequest(
				http.MethodGet,
				FMTConfigURL(
					"http", "localhost",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")).String(),
					"Bagelbone",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				),
				nil,
			)
			sig := model.NewRequestSignature(req, []byte("test"))
			sig.SetExpire(time.Now().Add(-time.Second))
			sig.PresignURL()
			return req
		}(),
		App: new(mapp.App),

		StatusCode: http.StatusForbidden,
		Error:      model.ErrLinkExpired,
	}, {
		Name: "error, signature invalid",

		Config: NewConfig().
			SetPresignSecret([]byte("test")),
		Request: func() *http.Request {
			req, _ := http.NewRequest(
				http.MethodGet,
				FMTConfigURL(
					"http", "localhost",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")).String(),
					"Bagelbone",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				),
				nil,
			)
			sig := model.NewRequestSignature(req, []byte("wrong_key"))
			sig.SetExpire(time.Now().Add(time.Minute))
			sig.PresignURL()
			return req
		}(),
		App: new(mapp.App),

		StatusCode: http.StatusForbidden,
		Error:      errors.New("signature invalid"),
	}, {
		Name: "error, deployment not found",

		Config: NewConfig().
			SetPresignSecret([]byte("test")),
		Request: func() *http.Request {
			req, _ := http.NewRequest(
				http.MethodGet,
				FMTConfigURL(
					"http", "localhost",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")).String(),
					"Bagelbone",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				),
				nil,
			)
			sig := model.NewRequestSignature(req, []byte("test"))
			sig.SetExpire(time.Now().Add(time.Minute))
			sig.PresignURL()
			return req
		}(),
		App: func() *mapp.App {
			appl := new(mapp.App)
			appl.On("GenerateConfigurationImage",
				contextMatcher(),
				"Bagelbone",
				uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")).String(),
			).Return(nil, app.ErrModelDeploymentNotFound)
			return appl
		}(),

		StatusCode: http.StatusNotFound,
		Error: errors.Errorf(
			"deployment with id '%s' not found",
			uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")),
		),
	}, {
		Name: "error, internal error",

		Config: NewConfig().
			SetPresignSecret([]byte("test")),
		Request: func() *http.Request {
			req, _ := http.NewRequest(
				http.MethodGet,
				FMTConfigURL(
					"http", "localhost",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")).String(),
					"Bagelbone",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				),
				nil,
			)
			sig := model.NewRequestSignature(req, []byte("test"))
			sig.SetExpire(time.Now().Add(time.Minute))
			sig.PresignURL()
			return req
		}(),
		App: func() *mapp.App {
			appl := new(mapp.App)
			appl.On("GenerateConfigurationImage",
				contextMatcher(),
				"Bagelbone",
				uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")).String(),
			).Return(nil, errors.New("internal error"))
			return appl
		}(),

		StatusCode: http.StatusInternalServerError,
		Error:      errors.New("internal error"),
	}, {
		Name: "error, broken artifact reader",

		Config: NewConfig().
			SetPresignSecret([]byte("test")),
		Request: func() *http.Request {
			req, _ := http.NewRequest(
				http.MethodGet,
				FMTConfigURL(
					"http", "localhost",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")).String(),
					"Bagelbone",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				),
				nil,
			)
			sig := model.NewRequestSignature(req, []byte("test"))
			sig.SetExpire(time.Now().Add(time.Minute))
			sig.PresignURL()
			return req
		}(),
		App: func() *mapp.App {
			appl := new(mapp.App)
			appl.On("GenerateConfigurationImage",
				contextMatcher(),
				"Bagelbone",
				uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")).String(),
			).Return(brokenReader{}, nil)
			return appl
		}(),

		StatusCode: http.StatusOK,
		Body:       []byte(nil),
	}}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			defer tc.App.AssertExpectations(t)
			reqClone := tc.Request.Clone(context.Background())
			handlers := NewDeploymentsApiHandlers(nil, &view.RESTView{}, tc.App, tc.Config)
			routes := NewDeploymentsResourceRoutes(handlers)
			router, _ := rest.MakeRouter(routes...)
			api := rest.NewApi()
			api.SetApp(router)
			handler := api.MakeHandler()
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, tc.Request)

			assert.Equal(t, tc.StatusCode, w.Code)
			if tc.Error != nil {
				var apiErr rest_utils.ApiError
				err := json.Unmarshal(w.Body.Bytes(), &apiErr)
				if assert.NoError(t, err) {
					assert.EqualError(t, &apiErr, tc.Error.Error())
				}
			} else {
				assert.Equal(t, w.Body.Bytes(), tc.Body)
				model.NewRequestSignature(reqClone, []byte("test"))
				rspHdr := w.Header()
				for key := range tc.Headers {
					if assert.Contains(t,
						rspHdr,
						key,
						"missing expected header",
					) {
						assert.Equal(t,
							tc.Headers.Get(key),
							rspHdr.Get(key),
						)
					}
				}
			}
		})
	}
}

func TestGetDeploymentForDevice(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name string

		Request  *http.Request
		App      *mapp.App
		IsConfig bool

		StatusCode int
		Error      error
	}{{
		Name: "ok",

		Request: func() *http.Request {
			req, _ := http.NewRequestWithContext(
				identity.WithContext(context.Background(), &identity.Identity{
					Subject:  uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
					IsDevice: true,
				}),
				http.MethodGet,
				"http://localhost"+ApiUrlDevicesDeploymentsNext+
					"?device_type=bagelShins&artifact_name=bagelOS1.0.1",
				nil,
			)
			return req
		}(),
		App: func() *mapp.App {
			app := new(mapp.App)
			app.On("GetDeploymentForDeviceWithCurrent",
				contextMatcher(),
				uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				&model.InstalledDeviceDeployment{
					ArtifactName: "bagelOS1.0.1",
					DeviceType:   "bagelShins",
				},
			).Return(&model.DeploymentInstructions{
				ID: uuid.NewSHA1(uuid.NameSpaceURL, []byte("deployment")).String(),
				Artifact: model.ArtifactDeploymentInstructions{
					ArtifactName:          "bagelOS1.1.0",
					DeviceTypesCompatible: []string{"bagelShins", "raspberryPlanck"},
					Source: model.Link{
						Uri:    "https://localhost/bucket/head/bagelOS1.0.1",
						Expire: time.Now().Add(time.Hour),
					},
				},
			}, nil)
			return app
		}(),

		StatusCode: http.StatusOK,
		Error:      nil,
	}, {
		Name: "ok, configuration deployment",

		Request: func() *http.Request {
			req, _ := http.NewRequestWithContext(
				identity.WithContext(context.Background(), &identity.Identity{
					Subject:  uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
					IsDevice: true,
				}),
				http.MethodGet,
				"http://localhost"+ApiUrlDevicesDeploymentsNext+
					"?device_type=bagelShins&artifact_name=bagelOS1.0.1",
				nil,
			)
			return req
		}(),
		App: func() *mapp.App {
			app := new(mapp.App)
			app.On("GetDeploymentForDeviceWithCurrent",
				contextMatcher(),
				uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				&model.InstalledDeviceDeployment{
					ArtifactName: "bagelOS1.0.1",
					DeviceType:   "bagelShins",
				},
			).Return(&model.DeploymentInstructions{
				ID: uuid.NewSHA1(uuid.NameSpaceURL, []byte("deployment")).String(),
				Artifact: model.ArtifactDeploymentInstructions{
					ArtifactName:          "bagelOS1.1.0",
					DeviceTypesCompatible: []string{"bagelShins", "raspberryPlanck"},
				},
				Type: model.DeploymentTypeConfiguration,
			}, nil)
			return app
		}(),
		IsConfig: true,

		StatusCode: http.StatusOK,
		Error:      nil,
	}, {
		Name: "ok, configuration deployment w/tenant",

		Request: func() *http.Request {
			req, _ := http.NewRequestWithContext(
				identity.WithContext(context.Background(), &identity.Identity{
					Subject:  uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
					IsDevice: true,
					Tenant:   "12456789012345678901234",
				}),
				http.MethodGet,
				"http://localhost"+ApiUrlDevicesDeploymentsNext+
					"?device_type=bagelShins&artifact_name=bagelOS1.0.1",
				nil,
			)
			return req
		}(),
		App: func() *mapp.App {
			app := new(mapp.App)
			app.On("GetDeploymentForDeviceWithCurrent",

				contextMatcher(),
				uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				&model.InstalledDeviceDeployment{
					ArtifactName: "bagelOS1.0.1",
					DeviceType:   "bagelShins",
				},
			).Return(&model.DeploymentInstructions{
				ID: uuid.NewSHA1(uuid.NameSpaceURL, []byte("deployment")).String(),
				Artifact: model.ArtifactDeploymentInstructions{
					ArtifactName:          "bagelOS1.1.0",
					DeviceTypesCompatible: []string{"bagelShins", "raspberryPlanck"},
				},
				Type: model.DeploymentTypeConfiguration,
			}, nil)
			return app
		}(),
		IsConfig: true,

		StatusCode: http.StatusOK,
		Error:      nil,
	}, {
		Name: "error, missing identity",

		Request: func() *http.Request {
			req, _ := http.NewRequest(
				http.MethodGet,
				"http://localhost"+ApiUrlDevicesDeploymentsNext+
					"?device_type=bagelShins&artifact_name=bagelOS1.0.1",
				nil,
			)
			return req
		}(),
		App: new(mapp.App),

		StatusCode: http.StatusBadRequest,
		Error:      ErrMissingIdentity,
	}, {
		Name: "error, missing parameters",

		Request: func() *http.Request {
			req, _ := http.NewRequestWithContext(
				identity.WithContext(context.Background(), &identity.Identity{
					Subject:  uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
					IsDevice: true,
					Tenant:   "12456789012345678901234",
				}),
				http.MethodGet,
				"http://localhost"+ApiUrlDevicesDeploymentsNext,
				nil,
			)
			return req
		}(),
		App: new(mapp.App),

		StatusCode: http.StatusBadRequest,
		Error:      errors.New("artifact_name: cannot be blank; device_type: cannot be blank."),
	}, {
		Name: "error, internal app error",

		Request: func() *http.Request {
			req, _ := http.NewRequestWithContext(
				identity.WithContext(context.Background(), &identity.Identity{
					Subject:  uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
					IsDevice: true,
					Tenant:   "12456789012345678901234",
				}),
				http.MethodGet,
				"http://localhost"+ApiUrlDevicesDeploymentsNext+
					"?device_type=bagelShins&artifact_name=bagelOS1.0.1",
				nil,
			)
			return req
		}(),
		App: func() *mapp.App {
			app := new(mapp.App)
			app.On("GetDeploymentForDeviceWithCurrent",

				contextMatcher(),
				uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				&model.InstalledDeviceDeployment{
					ArtifactName: "bagelOS1.0.1",
					DeviceType:   "bagelShins",
				},
			).Return(nil, errors.New("mongo: internal error"))
			return app
		}(),

		StatusCode: http.StatusInternalServerError,
		Error:      errors.New("internal error"),
	}, {
		Name: "error, internal app error",

		Request: func() *http.Request {
			req, _ := http.NewRequestWithContext(
				identity.WithContext(context.Background(), &identity.Identity{
					Subject:  uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
					IsDevice: true,
					Tenant:   "12456789012345678901234",
				}),
				http.MethodGet,
				"http://localhost"+ApiUrlDevicesDeploymentsNext+
					"?device_type=bagelShins&artifact_name=bagelOS1.0.1",
				nil,
			)
			return req
		}(),
		App: func() *mapp.App {
			app := new(mapp.App)
			app.On("GetDeploymentForDeviceWithCurrent",

				contextMatcher(),
				uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				&model.InstalledDeviceDeployment{
					ArtifactName: "bagelOS1.0.1",
					DeviceType:   "bagelShins",
				},
			).Return(nil, nil)
			return app
		}(),

		StatusCode: http.StatusNoContent,
	}}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			defer tc.App.AssertExpectations(t)
			config := NewConfig().
				SetPresignScheme("https").
				SetPresignHostname("localhost").
				SetPresignSecret([]byte("test")).
				SetPresignExpire(time.Hour)
			handlers := NewDeploymentsApiHandlers(nil, &view.RESTView{}, tc.App, config)
			routes := NewDeploymentsResourceRoutes(handlers)
			router, _ := rest.MakeRouter(routes...)
			api := rest.NewApi()
			api.SetApp(router)
			handler := api.MakeHandler()
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, tc.Request)

			assert.Equal(t, tc.StatusCode, w.Code)
			if tc.Error != nil {
				var apiErr rest_utils.ApiError
				err := json.Unmarshal(w.Body.Bytes(), &apiErr)
				if assert.NoError(t, err) {
					assert.EqualError(t, &apiErr, tc.Error.Error())
				}
			} else if tc.StatusCode == 204 {
				assert.Equal(t, []byte(nil), w.Body.Bytes())
			} else {
				if !assert.NotNil(t, w.Body.Bytes()) {
					return
				}
				var instr model.DeploymentInstructions
				json.Unmarshal(w.Body.Bytes(), &instr) //nolint: errcheck
				link, err := url.Parse(instr.Artifact.Source.Uri)
				if tc.IsConfig {
					assert.NoError(t, err)
					assert.Equal(t, "https", link.Scheme)
					assert.Equal(t, "localhost", link.Host)
					q := link.Query()
					expire, err := time.Parse(time.RFC3339, q.Get(model.ParamExpire))
					if assert.NoError(t, err) {
						assert.WithinDuration(t, time.Now().Add(time.Hour), expire, time.Minute)
					}
				}
				assert.WithinDuration(t, time.Now().Add(time.Hour), instr.Artifact.Source.Expire, time.Minute)
			}
		})
	}
}
