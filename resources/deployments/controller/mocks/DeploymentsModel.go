// Copyright 2016 Mender Software AS
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

package mocks

import context "context"
import controller "github.com/mendersoftware/deployments/resources/deployments/controller"
import deployments "github.com/mendersoftware/deployments/resources/deployments"
import mock "github.com/stretchr/testify/mock"

// DeploymentsModel is an autogenerated mock type for the DeploymentsModel type
type DeploymentsModel struct {
	mock.Mock
}

// AbortDeployment provides a mock function with given fields: deploymentID
func (_m *DeploymentsModel) AbortDeployment(deploymentID string) error {
	ret := _m.Called(deploymentID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(deploymentID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateDeployment provides a mock function with given fields: ctx, constructor
func (_m *DeploymentsModel) CreateDeployment(ctx context.Context, constructor *deployments.DeploymentConstructor) (string, error) {
	ret := _m.Called(ctx, constructor)

	var r0 string
	if rf, ok := ret.Get(0).(func(context.Context, *deployments.DeploymentConstructor) string); ok {
		r0 = rf(ctx, constructor)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *deployments.DeploymentConstructor) error); ok {
		r1 = rf(ctx, constructor)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DecommissionDevice provides a mock function with given fields: deviceID
func (_m *DeploymentsModel) DecommissionDevice(deviceID string) error {
	ret := _m.Called(deviceID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(deviceID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetDeployment provides a mock function with given fields: deploymentID
func (_m *DeploymentsModel) GetDeployment(deploymentID string) (*deployments.Deployment, error) {
	ret := _m.Called(deploymentID)

	var r0 *deployments.Deployment
	if rf, ok := ret.Get(0).(func(string) *deployments.Deployment); ok {
		r0 = rf(deploymentID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*deployments.Deployment)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(deploymentID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetDeploymentForDeviceWithCurrent provides a mock function with given fields: deviceID, current
func (_m *DeploymentsModel) GetDeploymentForDeviceWithCurrent(deviceID string, current deployments.InstalledDeviceDeployment) (*deployments.DeploymentInstructions, error) {
	ret := _m.Called(deviceID, current)

	var r0 *deployments.DeploymentInstructions
	if rf, ok := ret.Get(0).(func(string, deployments.InstalledDeviceDeployment) *deployments.DeploymentInstructions); ok {
		r0 = rf(deviceID, current)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*deployments.DeploymentInstructions)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, deployments.InstalledDeviceDeployment) error); ok {
		r1 = rf(deviceID, current)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetDeploymentStats provides a mock function with given fields: deploymentID
func (_m *DeploymentsModel) GetDeploymentStats(deploymentID string) (deployments.Stats, error) {
	ret := _m.Called(deploymentID)

	var r0 deployments.Stats
	if rf, ok := ret.Get(0).(func(string) deployments.Stats); ok {
		r0 = rf(deploymentID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(deployments.Stats)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(deploymentID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetDeviceDeploymentLog provides a mock function with given fields: deviceID, deploymentID
func (_m *DeploymentsModel) GetDeviceDeploymentLog(deviceID string, deploymentID string) (*deployments.DeploymentLog, error) {
	ret := _m.Called(deviceID, deploymentID)

	var r0 *deployments.DeploymentLog
	if rf, ok := ret.Get(0).(func(string, string) *deployments.DeploymentLog); ok {
		r0 = rf(deviceID, deploymentID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*deployments.DeploymentLog)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(deviceID, deploymentID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetDeviceStatusesForDeployment provides a mock function with given fields: deploymentID
func (_m *DeploymentsModel) GetDeviceStatusesForDeployment(deploymentID string) ([]deployments.DeviceDeployment, error) {
	ret := _m.Called(deploymentID)

	var r0 []deployments.DeviceDeployment
	if rf, ok := ret.Get(0).(func(string) []deployments.DeviceDeployment); ok {
		r0 = rf(deploymentID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]deployments.DeviceDeployment)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(deploymentID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// HasDeploymentForDevice provides a mock function with given fields: deploymentID, deviceID
func (_m *DeploymentsModel) HasDeploymentForDevice(deploymentID string, deviceID string) (bool, error) {
	ret := _m.Called(deploymentID, deviceID)

	var r0 bool
	if rf, ok := ret.Get(0).(func(string, string) bool); ok {
		r0 = rf(deploymentID, deviceID)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(deploymentID, deviceID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// IsDeploymentFinished provides a mock function with given fields: deploymentID
func (_m *DeploymentsModel) IsDeploymentFinished(deploymentID string) (bool, error) {
	ret := _m.Called(deploymentID)

	var r0 bool
	if rf, ok := ret.Get(0).(func(string) bool); ok {
		r0 = rf(deploymentID)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(deploymentID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// LookupDeployment provides a mock function with given fields: query
func (_m *DeploymentsModel) LookupDeployment(query deployments.Query) ([]*deployments.Deployment, error) {
	ret := _m.Called(query)

	var r0 []*deployments.Deployment
	if rf, ok := ret.Get(0).(func(deployments.Query) []*deployments.Deployment); ok {
		r0 = rf(query)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*deployments.Deployment)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(deployments.Query) error); ok {
		r1 = rf(query)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SaveDeviceDeploymentLog provides a mock function with given fields: deviceID, deploymentID, logs
func (_m *DeploymentsModel) SaveDeviceDeploymentLog(deviceID string, deploymentID string, logs []deployments.LogMessage) error {
	ret := _m.Called(deviceID, deploymentID, logs)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, []deployments.LogMessage) error); ok {
		r0 = rf(deviceID, deploymentID, logs)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateDeviceDeploymentStatus provides a mock function with given fields: deploymentID, deviceID, status
func (_m *DeploymentsModel) UpdateDeviceDeploymentStatus(deploymentID string, deviceID string, status string) error {
	ret := _m.Called(deploymentID, deviceID, status)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string) error); ok {
		r0 = rf(deploymentID, deviceID, status)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

var _ controller.DeploymentsModel = (*DeploymentsModel)(nil)
