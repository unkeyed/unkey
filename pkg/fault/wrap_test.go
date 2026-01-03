package fault

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
)

func TestWrapNil(t *testing.T) {
	result := Wrap(nil)
	require.Nil(t, result)
}

func TestWrapLocation(t *testing.T) {
	err := errors.New("base error")
	result := Wrap(err)

	wrapped, ok := result.(*wrapped)
	require.True(t, ok)
	require.NotEmpty(t, wrapped.location)
	require.Equal(t, err, wrapped.err)
}

func TestWrapSingleWrapper(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		wrapper        Wrapper
		expectedErr    error
		expectedCode   codes.URN
		expectedPublic string
		expectedInt    string
	}{
		{
			name:           "with internal description",
			err:            errors.New("base error"),
			wrapper:        Internal("internal message"),
			expectedErr:    errors.New("base error"),
			expectedCode:   "",
			expectedInt:    "internal message",
			expectedPublic: "",
		},
		{
			name:           "with public description",
			err:            errors.New("base error"),
			wrapper:        Public("public message"),
			expectedErr:    errors.New("base error"),
			expectedCode:   "",
			expectedInt:    "",
			expectedPublic: "public message",
		},
		{
			name:           "with code",
			err:            errors.New("base error"),
			wrapper:        Code(codes.URN("TEST_TAG")),
			expectedErr:    errors.New("base error"),
			expectedCode:   codes.URN("TEST_TAG"),
			expectedPublic: "",
			expectedInt:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log("ABC")
			result := Wrap(tt.err, tt.wrapper)

			wrapped, ok := result.(*wrapped)

			require.True(t, ok)
			require.NotNil(t, wrapped)
			t.Log("DEF")

			require.Equal(t, tt.expectedErr.Error(), wrapped.err.Error())
			t.Log("GHI")

			require.Equal(t, tt.expectedCode, wrapped.code)
			require.Equal(t, tt.expectedInt, wrapped.internal)
			require.Equal(t, tt.expectedPublic, wrapped.public)

		})
	}
}

func TestWrapMultipleWrappers(t *testing.T) {
	err := New("base error")
	err = Wrap(err,
		Internal("internal 1"),
		Public("public 1"),
		Code(codes.URN("TEST_TAG")),
	)
	err = Wrap(err,
		Internal("internal 2"),
		Public("public 2"),
	)

	// Verify base error is preserved
	require.Equal(t, "internal 2: internal 1: base error", err.Error())
}

func TestInternalNil(t *testing.T) {
	wrapper := Internal("internal")
	result := wrapper(nil)
	require.Nil(t, result)
}

func TestPublicNil(t *testing.T) {
	wrapper := Public("public")
	result := wrapper(nil)
	require.Nil(t, result)
}

func TestCodeNil(t *testing.T) {
	wrapper := Code(codes.URN("TEST"))
	result := wrapper(nil)
	require.Nil(t, result)
}

func TestNewAPIChaining(t *testing.T) {
	baseErr := errors.New("base error")
	first := Wrap(baseErr, Internal("internal 1"), Public("public 1"))
	second := Wrap(first, Internal("internal 2"), Public("public 2"))

	// Test error message includes internal descriptions
	require.Equal(t, "internal 2: internal 1: base error", second.Error())

	// Test user facing message includes public descriptions
	require.Equal(t, "public 2 public 1", UserFacingMessage(second))
}

func TestBasicWrapperFunctionality(t *testing.T) {
	baseErr := errors.New("base error")

	tests := []struct {
		name     string
		wrapper  Wrapper
		checkErr func(*testing.T, error)
	}{
		{
			name:    "Internal wrapper",
			wrapper: Internal("debug info"),
			checkErr: func(t *testing.T, err error) {
				w, ok := err.(*wrapped)
				require.True(t, ok)
				require.Equal(t, "debug info", w.internal)
				require.Equal(t, "", w.public)
			},
		},
		{
			name:    "Public wrapper",
			wrapper: Public("user message"),
			checkErr: func(t *testing.T, err error) {
				w, ok := err.(*wrapped)
				require.True(t, ok)
				require.Equal(t, "", w.internal)
				require.Equal(t, "user message", w.public)
			},
		},
		{
			name:    "Code wrapper",
			wrapper: Code(codes.URN("TEST_CODE")),
			checkErr: func(t *testing.T, err error) {
				w, ok := err.(*wrapped)
				require.True(t, ok)
				require.Equal(t, codes.URN("TEST_CODE"), w.code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.wrapper(baseErr)
			tt.checkErr(t, result)
		})
	}
}

func TestComplexErrorChaining(t *testing.T) {
	baseErr := errors.New("base error")
	err := Wrap(baseErr,
		Code(codes.URN("TEST_CODE")),
		Internal("internal message"),
		Public("public message"),
	)

	// Test error message includes internal descriptions only
	expectedError := "internal message: base error"
	require.Equal(t, expectedError, err.Error())

	// Test user facing message includes public descriptions
	expectedPublic := "public message"
	require.Equal(t, expectedPublic, UserFacingMessage(err))

	// Test code is preserved
	code, ok := GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.URN("TEST_CODE"), code)
}

func TestSingleWrappedInstance(t *testing.T) {
	baseErr := errors.New("base error")
	err := Wrap(baseErr,
		Code(codes.URN("TEST_CODE")),
		Internal("internal message"),
		Public("public message"),
	)

	// Verify we have only one wrapped instance
	wrappedErr, ok := err.(*wrapped)
	require.True(t, ok)

	// Check that all fields are set on the single instance
	require.Equal(t, codes.URN("TEST_CODE"), wrappedErr.code)
	require.Equal(t, "internal message", wrappedErr.internal)
	require.Equal(t, "public message", wrappedErr.public)
	require.Equal(t, baseErr, wrappedErr.err)

	// Verify there's no nested wrapped instances
	// The underlying error should be the original base error, not another wrapped
	require.Equal(t, baseErr, wrappedErr.err)

	// Test multiple messages accumulate properly in single instance
	multiErr := Wrap(baseErr,
		Internal("debug 1"),
		Public("user 1"),
		Internal("debug 2"),
		Public("user 2"),
		Code(codes.URN("MULTI_CODE")),
	)

	multiWrapped, ok := multiErr.(*wrapped)
	require.True(t, ok)

	// Verify messages are accumulated (newer first)
	require.Equal(t, "debug 2: debug 1", multiWrapped.internal)
	require.Equal(t, "user 2 user 1", multiWrapped.public)
	require.Equal(t, codes.URN("MULTI_CODE"), multiWrapped.code)
	require.Equal(t, baseErr, multiWrapped.err)
}
