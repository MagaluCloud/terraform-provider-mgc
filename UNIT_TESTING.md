# Unit Testing Guide for MGC Provider

This document describes the recommended approach for writing unit tests for **Resources** and **Data Sources** in the Terraform Provider MGC, using `terraform-plugin-framework` and the `testify/mock` library.

The main advantage of this approach is the ability to test provider logic (data mapping, error handling, etc.) in isolation, in milliseconds, without the need to create real infrastructure in the cloud (which would be the role of Acceptance Tests / `acctest`).

To run only unit tests (excluding acceptance tests):

```bash
go test ./mgc/... -short
```

## 1. General Approach

Instead of testing Terraform's complete lifecycle, we **instantiate the Resource/Data Source directly**, **inject a "Mocked" SDK (fake)** and manually invoke the `Create`, `Read`, `Update`, and `Delete` methods, passing and inspecting the `tfsdk.State` and `tfsdk.Plan` structures.

## 2. Generating SDK Mocks with Mockery

Instead of manually creating mock structs with `testify/mock`, we use **mockery** to dynamically generate these implementations based on Magalu Cloud SDK interfaces.

Add a `go:generate` directive at the top of the relevant source file (e.g., `resource_keys.go`):

```go
//go:generate go run github.com/vektra/mockery/v2@latest --name=ServiceName --srcpkg=github.com/MagaluCloud/mgc-sdk-go/package --output=./mgc/internal/mocks --outpkg=mocks
```

Then regenerate all mocks from the project root with:

```bash
go generate ./...
```

This will automatically create the file in the `mgc/internal/mocks` folder, allowing you to instantiate the mock cleanly in tests: `mockSvc := new(mocks.ServiceName)`.

## 3. The Schema Helper (`testutils`)

The `terraform-plugin-framework` requires a valid `schema.Schema` to instantiate and manipulate the `State`, `Plan`, or `Config`.

To ensure your tests use the real attribute definition of your resource without duplicating code, the project has generic utility packages in `mgc/internal/testutils`.

You should use the functions:

- `testutils.GetResourceTestSchema(t, r)` for Resources.
- `testutils.GetDataSourceTestSchema(t, d)` for Data Sources.

Both functions call `t.Fatal` on failure, so they act as hard stops — your test will not proceed with a broken schema.

## 4. Anatomy of a Unit Test

### 4.1 Basic Structure (Example: `Read`)

Below is the skeleton for structuring a unit test for provider methods. It is divided into 5 main steps:

```go
func TestYourResource_Read(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Step 1: Configure Mock Expectation
	// Define what we expect the Provider to call on the SDK and what the SDK should return.
	mockSvc := new(mocks.YourService)
	mockSvc.On("Get", ctx, "TEST_ID").Return(&sdkPackage.Model{
		ID:   "TEST_ID",
		Name: "new-api-name",
	}, nil)

	// Step 2: Initialize the Resource injecting the dependency (Mock)
	r := &yourResource{
		sdkClient: mockSvc,
	}

	// Step 3: Prepare the Input State/Plan
	// Use require for setup steps so the test stops immediately on failure.
	schemaResp := testutils.GetResourceTestSchema(t, r)
	state := tfsdk.State{Schema: schemaResp.Schema}

	inputData := yourProviderModel{
		ID:   types.StringValue("TEST_ID"),
		Name: types.StringValue("old-name"),
	}

	diags := state.Set(ctx, &inputData)
	require.False(t, diags.HasError(), "failed to set input state")

	// Step 4: Execute the Action and capture the Response
	req := resource.ReadRequest{State: state}
	resp := &resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Read(ctx, req, resp)

	// Step 5: Assertions (Validation)
	assert.False(t, resp.Diagnostics.HasError(), "Read should not return errors")

	var finalState yourProviderModel
	resp.State.Get(ctx, &finalState)

	assert.Equal(t, "TEST_ID", finalState.ID.ValueString())
	assert.Equal(t, "new-api-name", finalState.Name.ValueString())

	mockSvc.AssertExpectations(t)
}
```

> **`require` vs `assert`**: Use `require` for setup steps (schema loading, state population). A failure there means the test itself is broken and should stop. Use `assert` for the actual assertions you want to validate — they continue running even on failure, giving a fuller picture of what went wrong.

### 4.2 Table-Driven Tests

For testing multiple scenarios without duplicating the skeleton, use table-driven tests with `t.Run`:

```go
func TestYourResource_Read(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		inputID     string
		sdkResponse *sdkPackage.Model
		sdkError    error
		wantError   bool
		wantName    string
	}{
		{
			name:    "success",
			inputID: "TEST_ID",
			sdkResponse: &sdkPackage.Model{ID: "TEST_ID", Name: "api-name"},
			wantName: "api-name",
		},
		{
			name:      "sdk returns error",
			inputID:   "BAD_ID",
			sdkError:  errors.New("not found"),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			mockSvc := new(mocks.YourService)
			mockSvc.On("Get", ctx, tt.inputID).Return(tt.sdkResponse, tt.sdkError)

			r := &yourResource{sdkClient: mockSvc}
			schemaResp := testutils.GetResourceTestSchema(t, r)
			state := tfsdk.State{Schema: schemaResp.Schema}

			inputData := yourProviderModel{ID: types.StringValue(tt.inputID)}
			diags := state.Set(ctx, &inputData)
			require.False(t, diags.HasError())

			req := resource.ReadRequest{State: state}
			resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}

			r.Read(ctx, req, resp)

			if tt.wantError {
				assert.True(t, resp.Diagnostics.HasError())
			} else {
				assert.False(t, resp.Diagnostics.HasError())
				var finalState yourProviderModel
				resp.State.Get(ctx, &finalState)
				assert.Equal(t, tt.wantName, finalState.Name.ValueString())
			}

			mockSvc.AssertExpectations(t)
		})
	}
}
```

## 5. Differences for CRUD Methods

### Create (`resource.CreateRequest`)

Input data comes from the **Plan** (what the user defined in the `.tf` file), not from State. Attributes like `ID` are typically unknown at plan time.

```go
func TestYourResource_Create(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockSvc := new(mocks.YourService)
	mockSvc.On("Create", ctx, sdkPackage.CreateRequest{Name: "my-resource"}).
		Return(&sdkPackage.Model{ID: "new-id", Name: "my-resource"}, nil)

	r := &yourResource{sdkClient: mockSvc}
	schemaResp := testutils.GetResourceTestSchema(t, r)
	plan := tfsdk.Plan{Schema: schemaResp.Schema}

	inputData := yourProviderModel{
		ID:   types.StringUnknown(), // ID is unknown before creation
		Name: types.StringValue("my-resource"),
	}
	diags := plan.Set(ctx, &inputData)
	require.False(t, diags.HasError())

	req := resource.CreateRequest{Plan: plan}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}

	r.Create(ctx, req, resp)

	assert.False(t, resp.Diagnostics.HasError())

	var finalState yourProviderModel
	resp.State.Get(ctx, &finalState)
	assert.Equal(t, "new-id", finalState.ID.ValueString())
	assert.Equal(t, "my-resource", finalState.Name.ValueString())

	mockSvc.AssertExpectations(t)
}
```

### Update (`resource.UpdateRequest`)

Update receives both the current **State** (what exists) and the desired **Plan** (what the user wants). The response is a new State reflecting the applied changes.

```go
func TestYourResource_Update(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockSvc := new(mocks.YourService)
	mockSvc.On("Update", ctx, "existing-id", sdkPackage.UpdateRequest{Name: "new-name"}).
		Return(&sdkPackage.Model{ID: "existing-id", Name: "new-name"}, nil)

	r := &yourResource{sdkClient: mockSvc}
	schemaResp := testutils.GetResourceTestSchema(t, r)

	// Current state (what Terraform knows about)
	currentState := tfsdk.State{Schema: schemaResp.Schema}
	stateData := yourProviderModel{
		ID:   types.StringValue("existing-id"),
		Name: types.StringValue("old-name"),
	}
	diags := currentState.Set(ctx, &stateData)
	require.False(t, diags.HasError())

	// Desired plan (what the user wants)
	desiredPlan := tfsdk.Plan{Schema: schemaResp.Schema}
	planData := yourProviderModel{
		ID:   types.StringValue("existing-id"),
		Name: types.StringValue("new-name"),
	}
	diags = desiredPlan.Set(ctx, &planData)
	require.False(t, diags.HasError())

	req := resource.UpdateRequest{State: currentState, Plan: desiredPlan}
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}

	r.Update(ctx, req, resp)

	assert.False(t, resp.Diagnostics.HasError())

	var finalState yourProviderModel
	resp.State.Get(ctx, &finalState)
	assert.Equal(t, "existing-id", finalState.ID.ValueString())
	assert.Equal(t, "new-name", finalState.Name.ValueString())

	mockSvc.AssertExpectations(t)
}
```

### Delete (`resource.DeleteRequest`)

Input comes from **State**. Focus only on ensuring diagnostics has no errors and that the SDK Delete was called with the correct arguments — there is no output state to inspect.

```go
func TestYourResource_Delete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockSvc := new(mocks.YourService)
	mockSvc.On("Delete", ctx, "existing-id").Return(nil)

	r := &yourResource{sdkClient: mockSvc}
	schemaResp := testutils.GetResourceTestSchema(t, r)
	state := tfsdk.State{Schema: schemaResp.Schema}

	stateData := yourProviderModel{ID: types.StringValue("existing-id")}
	diags := state.Set(ctx, &stateData)
	require.False(t, diags.HasError())

	req := resource.DeleteRequest{State: state}
	resp := &resource.DeleteResponse{}

	r.Delete(ctx, req, resp)

	assert.False(t, resp.Diagnostics.HasError())
	mockSvc.AssertExpectations(t)
}
```

### Testing SDK Error Paths

Always test that when the SDK returns an error, the provider surfaces it correctly via `Diagnostics`:

```go
func TestYourResource_Delete_SDKError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockSvc := new(mocks.YourService)
	mockSvc.On("Delete", ctx, "existing-id").Return(errors.New("internal server error"))

	r := &yourResource{sdkClient: mockSvc}
	schemaResp := testutils.GetResourceTestSchema(t, r)
	state := tfsdk.State{Schema: schemaResp.Schema}

	stateData := yourProviderModel{ID: types.StringValue("existing-id")}
	diags := state.Set(ctx, &stateData)
	require.False(t, diags.HasError())

	req := resource.DeleteRequest{State: state}
	resp := &resource.DeleteResponse{}

	r.Delete(ctx, req, resp)

	assert.True(t, resp.Diagnostics.HasError(), "expected error to be propagated to diagnostics")
	mockSvc.AssertExpectations(t)
}
```

## 6. Testing Data Sources

The logic for testing Data Sources is similar, but with some important differences:

1. Use `testutils.GetDataSourceTestSchema(t, d)` to get the schema.
2. Input comes from **Config** (`tfsdk.Config`), not State.
3. `tfsdk.Config` does not have a `Set()` method, so you must manually build the `tftypes.Value`.

```go
func TestDataSourceSSH_Read(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockSvc := new(mocks.KeyService)
	mockSvc.On("List", ctx, sdkSSHKeys.ListOptions{}).Return([]sdkSSHKeys.SSHKey{
		{ID: "key-1", Name: "my-key-1", KeyType: "ssh-rsa"},
	}, nil)

	d := &DataSourceSSH{sshKeys: mockSvc}
	schemaResp := testutils.GetDataSourceTestSchema(t, d)

	// Build tftypes.Value manually for the Config (tfsdk.Config has no Set() method)
	sshKeysType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id": tftypes.String, "key_type": tftypes.String, "name": tftypes.String,
		},
	}
	listType := tftypes.List{ElementType: sshKeysType}

	configRaw := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{"ssh_keys": listType}},
		map[string]tftypes.Value{"ssh_keys": tftypes.NewValue(listType, []tftypes.Value{})},
	)

	config := tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}

	req := datasource.ReadRequest{Config: config}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}

	d.Read(ctx, req, resp)

	assert.False(t, resp.Diagnostics.HasError())

	var finalState SshKeysModel
	resp.State.Get(ctx, &finalState)

	assert.Len(t, finalState.SSHKeys, 1)
	assert.Equal(t, "key-1", finalState.SSHKeys[0].ID.ValueString())
	mockSvc.AssertExpectations(t)
}
```

## 7. Testing GetResources and GetDataSources

To test the functions that return the lists of resources and datasources from the package:

```go
func TestGetDataSources(t *testing.T) {
	t.Parallel()
	dataSources := GetDataSources()

	assert.NotEmpty(t, dataSources)

	for _, factory := range dataSources {
		ds := factory()
		assert.NotNil(t, ds)
		assert.Implements(t, (*datasource.DataSource)(nil), ds)
	}
}

func TestGetResources(t *testing.T) {
	t.Parallel()
	resources := GetResources()

	assert.NotEmpty(t, resources)

	for _, factory := range resources {
		r := factory()
		assert.NotNil(t, r)
		assert.Implements(t, (*resource.Resource)(nil), r)
	}
}
```

> **Note**: Avoid `assert.Len(t, resources, N)` with a hardcoded count — it breaks every time a new resource is added to the package for unrelated reasons. `assert.NotEmpty` is sufficient unless the exact count is specifically what you want to guard.

## 8. Test File Structure

Keep test files organized following the pattern:

- `resource_<name>_test.go` — Unit tests for Resources
- `datasource_<name>_test.go` — Unit tests for Data Sources
- `<package>_test.go` — Tests for package-level functions (`GetResources`, `GetDataSources`)

Example structure for the `ssh` package:

```
mgc/ssh/
├── datasource_keys.go          # DataSource implementation
├── datasource_keys_test.go     # DataSource tests
├── resource_keys.go            # Resource implementation (add //go:generate here)
├── resource_keys_test.go       # Resource tests
├── ssh.go                      # GetResources/GetDataSources functions
└── ssh_test.go                 # Package function tests
```
