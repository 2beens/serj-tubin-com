package file_box

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDiskApi(t *testing.T) {
	api, err := NewDiskApi("/var/invaliddir1234")
	assert.Error(t, err)
	assert.Nil(t, api)

	tempDir := t.TempDir()
	api, err = NewDiskApi(tempDir)
	require.NoError(t, err)
	assert.NotNil(t, api)
	assert.NotNil(t, api.root)
}

func TestDiskApi_UpdateInfo(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	api, err := NewDiskApi(tempDir)
	require.NoError(t, err)
	require.NotNil(t, api)

	// non-existing file
	file1Id := int64(100)
	parentId := int64(0)
	err = api.UpdateInfo(ctx, file1Id, "new-name", false)
	require.Error(t, err)
	assert.Equal(t, err.Error(), "file not found")

	fileName := "file1"
	randomContent := strings.NewReader("random test content")
	file1Id, err = api.Save(ctx, fileName, parentId, randomContent.Size(), "rand-binary", randomContent)
	require.NoError(t, err)
	assert.True(t, file1Id > 0)
	assert.Len(t, api.root.Files, 1)

	file1, ok := api.root.Files[file1Id]
	require.True(t, ok)
	require.NotNil(t, file1)

	// before update
	assert.Equal(t, fileName, file1.Name)
	assert.True(t, file1.IsPrivate)

	err = api.UpdateInfo(ctx, file1Id, "new-name", false)
	require.NoError(t, err)

	// after update
	assert.Equal(t, "new-name", file1.Name)
	assert.False(t, file1.IsPrivate)

	file1retrieved, parent, err := api.Get(ctx, file1Id)
	require.NoError(t, err)
	assert.Equal(t, "new-name", file1retrieved.Name)
	assert.False(t, file1retrieved.IsPrivate)
	assert.Equal(t, parentId, parent.Id)
}

func TestDiskApi_Save_InRoot(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	api, err := NewDiskApi(tempDir)
	require.NoError(t, err)
	require.NotNil(t, api)

	var addedFiles []int64
	parentId := int64(0) // root = id 0
	filesLen := 10
	for i := 1; i <= filesLen; i++ {
		randomContent := strings.NewReader(fmt.Sprintf("random test content %d", i))
		fileId, err := api.Save(
			ctx,
			fmt.Sprintf("file_%d", i),
			parentId,
			randomContent.Size(),
			"rand-binary",
			randomContent,
		)
		require.NoError(t, err)
		assert.True(t, fileId > 0)

		addedFiles = append(addedFiles, fileId)
	}
	assert.Len(t, api.root.Files, filesLen)
	require.Len(t, addedFiles, filesLen)

	file1, parent, err := api.Get(ctx, addedFiles[0])
	require.NoError(t, err)
	require.NotNil(t, file1)
	require.NotEmpty(t, file1.Path)
	assert.Equal(t, parentId, parent.Id)

	file1Content, err := os.ReadFile(file1.Path)
	require.NoError(t, err)
	assert.Equal(t, "random test content 1", string(file1Content))
}

func TestDiskApi_Save_InOtherFolder_ThenDelete(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	api, err := NewDiskApi(tempDir)
	require.NoError(t, err)
	require.NotNil(t, api)

	// add one file to the root
	randomContent := strings.NewReader("random test content 1")
	file1Id, err := api.Save(
		ctx,
		"file_1",
		0,
		randomContent.Size(),
		"rand-binary",
		randomContent,
	)
	require.NoError(t, err)
	require.True(t, file1Id > 0)
	assert.Len(t, api.root.Files, 1)

	folder1, err := api.NewFolder(context.Background(), 0, "folder1")
	require.NoError(t, err)
	require.NotNil(t, folder1)

	// add one file to the folder1
	randomContent = strings.NewReader("random test content 2")
	file2Id, err := api.Save(
		ctx,
		"file_2",
		folder1.Id,
		randomContent.Size(),
		"rand-binary",
		randomContent,
	)
	require.NoError(t, err)
	require.True(t, file2Id > 0)
	assert.Len(t, api.root.Files, 1)
	assert.Len(t, folder1.Files, 1)
	assert.Len(t, folder1.Subfolders, 0)

	retrievedFile2, _, err := api.Get(ctx, file2Id)
	require.NoError(t, err)
	require.NotNil(t, retrievedFile2)
	assert.Equal(t, file2Id, retrievedFile2.Id)
	assert.Equal(t, "file_2", retrievedFile2.Name)
	assert.True(t, retrievedFile2.IsPrivate)

	// now test delete
	err = api.Delete(context.Background(), 1000) // try delete non existing file
	assert.ErrorIs(t, err, ErrFileNotFound)

	err = api.Delete(context.Background(), file2Id)
	require.NoError(t, err)
	retrievedFile2, _, err = api.Get(ctx, file2Id)
	assert.ErrorIs(t, err, ErrFileNotFound)
	require.Nil(t, retrievedFile2)

	assert.Len(t, api.root.Files, 1)
	assert.Len(t, api.root.Subfolders, 1)
}

func TestDiskApi_DeleteFolder(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	api, err := NewDiskApi(tempDir)
	require.NoError(t, err)
	require.NotNil(t, api)

	err = api.DeleteFolder(context.Background(), 0)
	assert.Equal(t, "cannot delete root folder", err.Error())

	// add one file to the root
	randomContent := strings.NewReader("random test content 1")
	file1Id, err := api.Save(
		ctx,
		"file_1",
		0,
		randomContent.Size(),
		"rand-binary",
		randomContent,
	)
	require.NoError(t, err)
	require.True(t, file1Id > 0)
	assert.Len(t, api.root.Files, 1)

	folder1, err := api.NewFolder(context.Background(), 0, "folder1")
	require.NoError(t, err)
	require.NotNil(t, folder1)
	folder2, err := api.NewFolder(context.Background(), 0, "folder2")
	require.NoError(t, err)
	require.NotNil(t, folder2)

	// add one file to the folder1
	randomContent = strings.NewReader("random test content 2")
	file2Id, err := api.Save(
		ctx,
		"file_2",
		folder1.Id,
		randomContent.Size(),
		"rand-binary",
		randomContent,
	)
	require.NoError(t, err)
	require.True(t, file2Id > 0)
	assert.Len(t, api.root.Files, 1)
	assert.Len(t, folder1.Files, 1)
	assert.Len(t, folder1.Subfolders, 0)

	folder11, err := api.NewFolder(context.Background(), folder1.Id, "folder11")
	require.NoError(t, err)
	require.NotNil(t, folder11)

	assert.Len(t, api.root.Files, 1)
	assert.Len(t, folder1.Files, 1)
	assert.Len(t, folder1.Subfolders, 1)

	err = api.DeleteFolder(context.Background(), 1000) // non existent folder
	require.ErrorIs(t, err, ErrFolderNotFound)

	err = api.DeleteFolder(context.Background(), folder1.Id)
	require.NoError(t, err)
	assert.Len(t, api.root.Files, 1)
	assert.Len(t, api.root.Subfolders, 1) // only folder2 left in the root
	folder1, err = api.GetFolder(ctx, folder1.Id)
	require.ErrorIs(t, err, ErrFolderNotFound)
	assert.Nil(t, folder1)
	folder11, err = api.GetFolder(ctx, folder11.Id)
	require.ErrorIs(t, err, ErrFolderNotFound)
	assert.Nil(t, folder11)
}
