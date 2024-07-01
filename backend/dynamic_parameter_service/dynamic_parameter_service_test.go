// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

// +build ui

package dynamic_parameter_service

import (
	"io"
	"os"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

const (
	CONFIG_FILE_NAME = "test_file.txt"
	TEST_ENV_VAR = "TEST_ENV_VAR"
	EXPECTED_FILE_CONTENT = "Some Config Content"
)

func writeTestFile(t *testing.T, content string) {
	err := os.WriteFile(CONFIG_FILE_NAME, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
}

func checkConfigFileContent(t *testing.T, testServer *httptest.Server, expectedFileContent string){
	resp, _ := http.Get(testServer.URL + "/config/route")
	if resp.StatusCode != http.StatusOK {
        t.Errorf("expected status OK, got %d", resp.StatusCode)
    }

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Failed to read file content")
	}
	if string(body) != expectedFileContent {
        t.Errorf("expected body '%s', got '%s'", expectedFileContent, string(body))
    }
}

func checkConfigFileIsNotServed(t *testing.T, testServer *httptest.Server){
	resp, _ := http.Get(testServer.URL + "/config/route")
	if resp.StatusCode != http.StatusNotFound {
        t.Errorf("expected status 404, got %d", resp.StatusCode)
    }
}

func TestGivenEnvironmentVariableAndFileSetWhenServeDynamicParameterThenFileContentIsServed(t *testing.T) {
	os.Setenv(TEST_ENV_VAR, CONFIG_FILE_NAME)
	writeTestFile(t, EXPECTED_FILE_CONTENT)
	defer func() {
		os.Remove(CONFIG_FILE_NAME)
	}()

	engine := gin.Default()
	group := engine.Group("/config/")
	serveDynamicParameter(group, "route", TEST_ENV_VAR)
	testServer := httptest.NewServer(engine)
	defer testServer.Close()

	checkConfigFileContent(t, testServer, EXPECTED_FILE_CONTENT)
}

func TestGivenEnvironmentVariableIsNotSetWhenServeDynamicParameterThenFileContentIsNotServed(t *testing.T) {
	os.Setenv(TEST_ENV_VAR, "")
	writeTestFile(t, EXPECTED_FILE_CONTENT)
	defer func() {
		os.Remove(CONFIG_FILE_NAME)
	}()

	engine := gin.Default()
	group := engine.Group("/config/")
	serveDynamicParameter(group, "route", TEST_ENV_VAR)
	testServer := httptest.NewServer(engine)
	defer testServer.Close()

	checkConfigFileIsNotServed(t, testServer)
}

func TestGivenEnvironmentVariableIsSetButFileDoesNotExistWhenServeDynamicParameterThenFileContentIsNotServed(t *testing.T) {
	os.Setenv(TEST_ENV_VAR, CONFIG_FILE_NAME)

	engine := gin.Default()
	group := engine.Group("/config/")
	serveDynamicParameter(group, "route", TEST_ENV_VAR)
	testServer := httptest.NewServer(engine)
	defer testServer.Close()

	checkConfigFileIsNotServed(t, testServer)
}

func TestGivenConfigFileExistsButItIsEmptyWhenServeDynamicParameterThenFileContentIsServed(t *testing.T) {
	os.Setenv(TEST_ENV_VAR, CONFIG_FILE_NAME)
	writeTestFile(t, "")
	defer func() {
		os.Remove(CONFIG_FILE_NAME)
	}()

	engine := gin.Default()
	group := engine.Group("/config/")
	serveDynamicParameter(group, "route", TEST_ENV_VAR)
	testServer := httptest.NewServer(engine)
	defer testServer.Close()

	checkConfigFileContent(t, testServer, "")
}
