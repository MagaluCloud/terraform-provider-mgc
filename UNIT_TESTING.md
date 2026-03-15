# Unit Testing Guide for MGC Provider

This document describes the recommended approach for writing unit tests for **Resources** and **Data Sources** in the Terraform Provider MGC, using `terraform-plugin-framework` and the `testify/mock` library.

The main advantage of this approach is the ability to test provider logic (data mapping, error handling, etc.) in isolation, in milliseconds, without the need to create real infrastructure in the cloud (which would be the role of Acceptance Tests / `acctest`).

## 1. General Approach

Instead of testing Terraform's complete lifecycle, we **instantiate the Resource/Data Source directly**, **inject a "Mocked" SDK (fake)** and manually invoke the `Create`, `Read`, `Update`, and `Delete` methods, passing and inspecting the `tfsdk.State` and `tfsdk.Plan` structures.

## 2. Generating SDK Mocks with Mockery

Instead of manually creating mock structs with `testify/mock`, we use **mockery** to dynamically generate these implementations based on Magalu Cloud SDK interfaces.

To generate a mock for a new service, you can use the `mockery` command. Ideally, add a `go:generate` command and run `go generate ./...` or use the CLI directly at the project root:

```bash
go run github.com/vektra/mockery/v2@latest --name=ServiceName --srcpkg=github.com/MagaluCloud/mgc-sdk-go/package --output=./mgc/internal/mocks --outpkg=mocks
```

This will automatically create the file in the `mgc/internal/mocks` folder, allowing you to instantiate the mock cleanly in tests: `mockSvc := new(mocks.ServiceName)`.

## 3. The Schema Helper (`testutils`)

The `terraform-plugin-framework` requires a valid `schema.Schema` to instantiate and manipulate the `State`, `Plan`, or `Config`.

To ensure your tests use the real attribute definition of your resource without duplicating code, the project has generic utility packages in `mgc/internal/testutils`.

You should use the functions:

- `testutils.GetResourceTestSchema(t, r)` for Resources.
- `testutils.GetDataSourceTestSchema(t, d)` for Data Sources.

## 4. Anatomy of a Unit Test (Example: `Read` Method)

Below is the skeleton for structuring a unit test for provider methods. It is divided into 5 main steps:

```go
func TestYourResource_Read(t *testing.T) {
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
	schemaResp := testutils.GetResourceTestSchema(t, r)
	state := tfsdk.State{Schema: schemaResp.Schema}

	inputData := yourProviderModel{
		ID:   types.StringValue("TEST_ID"),
		Name: types.StringValue("old-name"),
	}

	diags := state.Set(ctx, &inputData)
	assert.False(t, diags.HasError())

	// Step 4: Execute the Action and capture the Response
	req := resource.ReadRequest{
		State: state,
	}
	resp := &resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Read(ctx, req, resp) // The magic happens here!

	// Step 5: Assertions (Validation)
	assert.False(t, resp.Diagnostics.HasError(), "Operation should not return errors")

	var finalState yourProviderModel
	resp.State.Get(ctx, &finalState) // Extract data saved in the new state

	// Verify that the Provider correctly mapped the API response to Terraform state
	assert.Equal(t, "TEST_ID", finalState.ID.ValueString())
	assert.Equal(t, "new-api-name", finalState.Name.ValueString())

	// Ensure the mocked SDK method was actually invoked
	mockSvc.AssertExpectations(t)
}
```

## 5. Testing Data Sources

The logic for testing Data Sources is similar, but with some important differences:

1. Use `testutils.GetDataSourceTestSchema(t, d)` to get the schema
2. Input comes from **Config** (`tfsdk.Config`), not State
3. `tfsdk.Config` does not have a `Set()` method, so you must manually build the `tftypes.Value`

```go
func TestDataSourceSSH_Read(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.KeyService)
	mockSvc.On("List", ctx, sdkSSHKeys.ListOptions{}).Return([]sdkSSHKeys.SSHKey{
		{ID: "key-1", Name: "my-key-1", KeyType: "ssh-rsa"},
	}, nil)

	d := &DataSourceSSH{sshKeys: mockSvc}
	schemaResp := testutils.GetDataSourceTestSchema(t, d)

	// Build tftypes.Value manually for the Config
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

## 6. Testing GetResources and GetDataSources

To test the functions that return the lists of resources and datasources from the package:

```go
func TestGetDataSources(t *testing.T) {
	dataSources := GetDataSources()

	assert.NotNil(t, dataSources)
	assert.Len(t, dataSources, 1)

	factory := dataSources[0]
	assert.NotNil(t, factory)

	ds := factory()
	assert.NotNil(t, ds)
	assert.Implements(t, (*datasource.DataSource)(nil), ds)

	_, ok := ds.(*DataSourceSSH)
	assert.True(t, ok)
}

func TestGetResources(t *testing.T) {
	resources := GetResources()

	assert.NotNil(t, resources)
	assert.Len(t, resources, 1)

	factory := resources[0]
	assert.NotNil(t, factory)

	r := factory()
	assert.NotNil(t, r)
	assert.Implements(t, (*resource.Resource)(nil), r)

	_, ok := r.(*sshKeys)
	assert.True(t, ok)
}
```

## 7. Differences for Other Methods

### Create (`resource.CreateRequest`)

- In the Create method, input data comes from the **Plan** (what the user defined in the `.tf` file), not from State.
- Build the Plan the same way you built the state: `plan := tfsdk.Plan{Schema: schemaResp.Schema}`.
- Call `plan.Set(ctx, &inputData)`. Remember that during creation planning, attributes like `ID` are usually unknown (`types.StringUnknown()`).

### Delete (`resource.DeleteRequest`)

- Input comes from **State**.
- There is usually no need to inspect an output state on a deleted resource, you focus only on ensuring that `resp.Diagnostics` has no errors and that the mocked `Delete` SDK was called with the correct ID.

## 8. Test File Structure

Keep test files organized following the pattern:

- `resource_<name>_test.go` - Unit tests for Resources
- `datasource_<name>_test.go` - Unit tests for Data Sources
- `ssh_test.go` - Tests for package auxiliary functions (like `GetResources`, `GetDataSources`)

Example structure for the `ssh` package:

```
mgc/ssh/
├── datasource_keys.go          # DataSource implementation
├── datasource_keys_test.go     # DataSource tests
├── resource_keys.go            # Resource implementation
├── resource_keys_test.go       # Resource tests
├── ssh.go                      # GetResources/GetDataSources functions
└── ssh_test.go                 # Package function tests
```
