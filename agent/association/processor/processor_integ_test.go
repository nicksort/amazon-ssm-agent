// Copyright 2016 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not
// use this file except in compliance with the License. A copy of the
// License is located at
//
// http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
// either express or implied. See the License for the specific language governing
// permissions and limitations under the License.

// Package processor manage polling of associations, dispatching association to processor
package processor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/aws/amazon-ssm-agent/agent/appconfig"
	"github.com/aws/amazon-ssm-agent/agent/association/model"
	"github.com/aws/amazon-ssm-agent/agent/context"
	"github.com/aws/amazon-ssm-agent/agent/contracts"
	"github.com/aws/amazon-ssm-agent/agent/log"
	messageContracts "github.com/aws/amazon-ssm-agent/agent/message/contracts"
	stateModel "github.com/aws/amazon-ssm-agent/agent/statemanager/model"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/stretchr/testify/assert"
)

const (
	FILE_VERSION_1_0 = "./testdata/sampleVersion1_0.json"
	FILE_VERSION_1_2 = "./testdata/sampleVersion1_2.json"
	FILE_VERSION_2_0 = "./testdata/sampleVersion2_0.json"
)

func TestParseAssociationWithAssociationVersion1_2(t *testing.T) {
	log := log.Logger()
	context := context.Default(log, appconfig.SsmagentConfig{})
	processor := NewAssociationProcessor(context, "i-test")
	sys = &systemStub{}

	sampleFile := readFile(FILE_VERSION_1_2)

	instanceID := "i-test"
	commandId := "commandV1.2"
	associationName := "testV1.2"
	assocRawData := model.AssociationRawData{
		ID:         "test-id",
		CreateDate: "2016-10-10",
		Document:   &sampleFile,
	}
	assocRawData.ID = commandId
	assocRawData.Association = &ssm.Association{}
	assocRawData.Association.Name = &associationName
	assocRawData.Association.InstanceId = &instanceID
	assocRawData.Parameter = &ssm.AssociationDescription{}

	params := make(map[string][]*string)
	address := "http://7-zip.org/a/7z1602-x64.msi"
	source := []*string{&address}
	params["source"] = source

	assocRawData.Parameter.Parameters = params

	docState, err := processor.parseAssociation(&assocRawData)

	documentInfo := new(stateModel.DocumentInfo)
	documentInfo.CommandID = commandId
	documentInfo.Destination = instanceID
	documentInfo.MessageID = fmt.Sprintf("aws.ssm.%v.%v", commandId, instanceID)
	documentInfo.DocumentName = associationName

	pluginName := "aws:applications"
	pluginsInfo := make(map[string]stateModel.PluginState)
	config := contracts.Configuration{}
	var plugin stateModel.PluginState
	plugin.Configuration = config
	plugin.HasExecuted = false
	plugin.Id = pluginName
	pluginsInfo[pluginName] = plugin

	expectedDocState := stateModel.DocumentState{
		//DocumentInformation: documentInfo,
		PluginsInformation: pluginsInfo,
		DocumentType:       stateModel.Association,
		SchemaVersion:      "1.2",
	}

	payload := &messageContracts.SendCommandPayload{}

	err2 := json.Unmarshal([]byte(*assocRawData.Document), &payload.DocumentContent)
	pluginConfig := payload.DocumentContent.RuntimeConfig[pluginName]

	assert.Equal(t, nil, err)
	assert.Equal(t, nil, err2)
	assert.Equal(t, expectedDocState.SchemaVersion, docState.SchemaVersion)
	assert.Equal(t, stateModel.Association, docState.DocumentType)
	assert.True(t, docState.InstancePluginsInformation == nil)

	pluginInfo := docState.PluginsInformation[pluginName]
	expectedProp := []interface{}{map[string]interface{}{"source": source[0], "sourceHash": "", "id": "0.aws:applications", "action": "Install", "parameters": ""}}

	assert.Equal(t, expectedProp, pluginInfo.Configuration.Properties)
	assert.Equal(t, pluginConfig.Settings, pluginInfo.Configuration.Settings)
	assert.Equal(t, documentInfo.MessageID, pluginInfo.Configuration.MessageId)
}

func TestParseAssociationWithAssociationVersion2_0(t *testing.T) {

	log := log.Logger()
	context := context.Default(log, appconfig.SsmagentConfig{})
	processor := NewAssociationProcessor(context, "i-test")
	sys = &systemStub{}

	sampleFile := readFile(FILE_VERSION_2_0)

	instanceID := "i-test"
	commandId := "commandV2.0"
	associationName := "testV2.0"
	assocRawData := model.AssociationRawData{
		ID:         "test-id",
		CreateDate: "2016-10-10",
		Document:   &sampleFile,
	}
	assocRawData.ID = commandId
	assocRawData.Association = &ssm.Association{}
	assocRawData.Association.Name = &associationName
	assocRawData.Association.InstanceId = &instanceID
	assocRawData.Parameter = &ssm.AssociationDescription{}

	params := make(map[string][]*string)
	cmd0 := "ls"
	source0 := []*string{&cmd0}
	cmd1 := "pwd"
	source1 := []*string{&cmd1}
	params["runCommand0"] = source0
	params["runCommand1"] = source1

	assocRawData.Parameter.Parameters = params

	// test the method
	docState, err := processor.parseAssociation(&assocRawData)

	documentInfo := new(stateModel.DocumentInfo)
	documentInfo.CommandID = commandId
	documentInfo.Destination = instanceID
	documentInfo.MessageID = fmt.Sprintf("aws.ssm.%v.%v", commandId, instanceID)
	documentInfo.DocumentName = associationName

	instancePluginsInfo := make([]stateModel.PluginState, 2)

	action0 := "aws:runPowerShellScript"
	name0 := "runPowerShellScript1"
	var plugin0 stateModel.PluginState
	plugin0.Configuration = contracts.Configuration{}
	plugin0.HasExecuted = false
	plugin0.Id = name0
	plugin0.Name = action0
	instancePluginsInfo[0] = plugin0

	action1 := "aws:runPowerShellScript"
	name1 := "runPowerShellScript2"
	var plugin1 stateModel.PluginState
	plugin1.Configuration = contracts.Configuration{}
	plugin1.HasExecuted = false
	plugin1.Id = name1
	plugin1.Name = action1
	instancePluginsInfo[1] = plugin1

	expectedDocState := stateModel.DocumentState{
		//DocumentInformation: documentInfo,
		InstancePluginsInformation: instancePluginsInfo,
		DocumentType:               stateModel.Association,
		SchemaVersion:              "2.0",
	}

	assert.Equal(t, nil, err)
	assert.True(t, docState.PluginsInformation == nil)
	assert.Equal(t, expectedDocState.SchemaVersion, docState.SchemaVersion)
	assert.Equal(t, stateModel.Association, docState.DocumentType)
	assert.Equal(t, documentInfo.MessageID, docState.DocumentInformation.MessageID)

	pluginInfo1 := docState.InstancePluginsInformation[0]
	pluginInfo2 := docState.InstancePluginsInformation[1]

	assert.Equal(t, name0, pluginInfo1.Id)
	assert.Equal(t, name1, pluginInfo2.Id)
	assert.Equal(t, action0, pluginInfo1.Name)
	assert.Equal(t, action1, pluginInfo2.Name)

	expectProp1 := map[string]interface{}{"id": "0.aws:psModule", "runCommand": source0[0]}
	expectProp2 := map[string]interface{}{"id": "1.aws:psModule", "runCommand": source1[0]}

	assert.Equal(t, expectProp1, pluginInfo1.Configuration.Properties)
	assert.Equal(t, expectProp2, pluginInfo2.Configuration.Properties)
}

func readFile(fileName string) string {
	file, e := ioutil.ReadFile(fileName)
	if e != nil {
		fmt.Printf("File error: %v\n", e)
		os.Exit(1)
	}
	return string(file)
}
