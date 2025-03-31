package fault

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
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
		expectedTag    Tag
		expectedPublic string
		expectedInt    string
	}{
		{
			name:           "with description",
			err:            errors.New("base error"),
			wrapper:        WithDesc("internal message", "public message"),
			expectedErr:    errors.New("base error"),
			expectedTag:    "",
			expectedInt:    "internal message",
			expectedPublic: "public message",
		},
		{
			name:           "with tag",
			err:            errors.New("base error"),
			wrapper:        WithTag(Tag("TEST_TAG")),
			expectedErr:    errors.New("base error"),
			expectedTag:    Tag("TEST_TAG"),
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

			require.Equal(t, tt.expectedTag, wrapped.tag)
			require.Equal(t, tt.expectedInt, wrapped.internal)
			require.Equal(t, tt.expectedPublic, wrapped.public)

		})
	}
}

func TestWrapMultipleWrappers(t *testing.T) {
	err := New("base error")
	err = Wrap(err,
		WithDesc("internal 1", "public 1"),
		WithTag(Tag("TEST_TAG")),
	)
	err = Wrap(err,
		WithDesc("internal 2", "public 2"),
	)

	// Verify base error is preserved
	require.Equal(t, "internal 2: internal 1: base error", err.Error())
}

func TestWithDescNil(t *testing.T) {
	wrapper := WithDesc("internal", "public")
	result := wrapper(nil)
	require.Nil(t, result)
}

func TestWithDescBasic(t *testing.T) {
	tests := []struct {
		name        string
		internal    string
		public      string
		baseErr     error
		expectedInt string
		expectedPub string
	}{
		{
			name:        "empty descriptions",
			internal:    "",
			public:      "",
			baseErr:     errors.New("base error"),
			expectedInt: "",
			expectedPub: "",
		},
		{
			name:        "normal descriptions",
			internal:    "internal desc",
			public:      "public desc",
			baseErr:     errors.New("base error"),
			expectedInt: "internal desc",
			expectedPub: "public desc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WithDesc(tt.internal, tt.public)(tt.baseErr)
			wrapped, ok := result.(*wrapped)
			require.True(t, ok)
			require.Equal(t, tt.expectedInt, wrapped.internal)
			require.Equal(t, tt.expectedPub, wrapped.public)
			require.Equal(t, tt.baseErr.Error(), wrapped.err.Error())
		})
	}
}

func TestWithDescChaining(t *testing.T) {
	baseErr := errors.New("base error")
	first := WithDesc("internal 1", "public 1")(baseErr)
	second := WithDesc("internal 2", "public 2")(first)

	w1, ok := second.(*wrapped)
	require.True(t, ok)
	require.Equal(t, "internal 2", w1.internal)
	require.Equal(t, "public 2", w1.public)

	w2, ok := w1.err.(*wrapped)
	require.True(t, ok)
	require.Equal(t, "internal 1", w2.internal)
	require.Equal(t, "public 1", w2.public)
	require.Equal(t, baseErr.Error(), w2.err.Error())
}
