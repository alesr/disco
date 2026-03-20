package review

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUnifiedDiff(t *testing.T) {
	t.Parallel()

	t.Run("parses single hunk", func(t *testing.T) {
		t.Parallel()

		diff := `diff --git a/main.go b/main.go
index 1111111..2222222 100644
--- a/main.go
+++ b/main.go
@@ -10,2 +10,3 @@
 fmt.Println("old")
+fmt.Println("new")
 return nil`

		hunks, err := ParseUnifiedDiff(diff)
		require.NoError(t, err)
		require.Len(t, hunks, 1)
		assert.Equal(t, "main.go", hunks[0].File)
		assert.Equal(t, 10, hunks[0].Line)
		assert.Contains(t, hunks[0].Content, "fmt.Println")
	})

	t.Run("parses multiple hunks", func(t *testing.T) {
		t.Parallel()

		diff := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,1 +1,1 @@
-a
+b
@@ -5,1 +5,1 @@
-x
+y`

		hunks, err := ParseUnifiedDiff(diff)
		require.NoError(t, err)
		require.Len(t, hunks, 2)
		assert.Equal(t, 1, hunks[0].Line)
		assert.Equal(t, 5, hunks[1].Line)
	})

	t.Run("empty diff returns error", func(t *testing.T) {
		t.Parallel()

		_, err := ParseUnifiedDiff("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "diff is empty")
	})

	t.Run("parses deletion hunk with zero start", func(t *testing.T) {
		t.Parallel()

		diff := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -12,4 +0,0 @@
-func old() {}
-func gone() {}`

		hunks, err := ParseUnifiedDiff(diff)
		require.NoError(t, err)
		require.Len(t, hunks, 1)
		assert.Equal(t, 0, hunks[0].Line)
	})
}
