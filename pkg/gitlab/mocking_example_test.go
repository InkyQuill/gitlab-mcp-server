package gitlab

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gl "gitlab.com/gitlab-org/api/client-go"
	gltesting "gitlab.com/gitlab-org/api/client-go/testing"
	"go.uber.org/mock/gomock"
)

// TestGitLabOfficialTesting_Example demonstrates how to use GitLab's official testing package
// This replaces our custom mocking framework with the official gomock-based solution
func TestGitLabOfficialTesting_Example(t *testing.T) {
	t.Run("Example - Mock CurrentUser", func(t *testing.T) {
		// Create a test client with mocked services
		client := gltesting.NewTestClient(t)

		// Setup expectations using gomock
		client.MockUsers.EXPECT().
			CurrentUser(gomock.Any()).
			Return(&gl.User{
				ID:       123,
				Username: "testuser",
				Name:     "Test User",
				Email:    "test@example.com",
			}, &gl.Response{
				Response: &http.Response{
					StatusCode: 200,
				},
			}, nil)

		// Use the client
		user, resp, err := client.Users.CurrentUser()
		require.NoError(t, err)
		assert.Equal(t, 123, user.ID)
		assert.Equal(t, "testuser", user.Username)
		assert.Equal(t, 200, resp.Response.StatusCode)
	})

	t.Run("Example - Mock GetProject", func(t *testing.T) {
		client := gltesting.NewTestClient(t)

		projectID := "mygroup/myproject"
		client.MockProjects.EXPECT().
			GetProject(projectID, gomock.Any()).
			Return(&gl.Project{
				ID:                456,
				Name:              "myproject",
				Path:              "myproject",
				PathWithNamespace: "mygroup/myproject",
			}, &gl.Response{
				Response: &http.Response{
					StatusCode: 200,
				},
			}, nil)

		project, resp, err := client.Projects.GetProject(projectID, nil)
		require.NoError(t, err)
		assert.Equal(t, 456, project.ID)
		assert.Equal(t, "myproject", project.Name)
		assert.Equal(t, 200, resp.Response.StatusCode)
	})

	t.Run("Example - Mock CreateIssue", func(t *testing.T) {
		client := gltesting.NewTestClient(t)

		projectID := "mygroup/myproject"
		client.MockIssues.EXPECT().
			CreateIssue(projectID, gomock.Any()).
			Return(&gl.Issue{
				IID:       1,
				Title:     "Test Issue",
				ProjectID: 456,
				State:     "opened",
			}, &gl.Response{
				Response: &http.Response{
					StatusCode: 201,
				},
			}, nil)

		issue, resp, err := client.Issues.CreateIssue(projectID, &gl.CreateIssueOptions{
			Title: gl.Ptr("Test Issue"),
		})
		require.NoError(t, err)
		assert.Equal(t, 1, issue.IID)
		assert.Equal(t, "Test Issue", issue.Title)
		assert.Equal(t, 201, resp.Response.StatusCode)
	})

	t.Run("Example - Mock Error Response", func(t *testing.T) {
		client := gltesting.NewTestClient(t)

		projectID := "nonexistent/project"
		client.MockProjects.EXPECT().
			GetProject(projectID, gomock.Any()).
			Return(nil, &gl.Response{
				Response: &http.Response{
					StatusCode: 404,
				},
			}, errors.New("404 Not Found"))

		project, resp, err := client.Projects.GetProject(projectID, nil)
		require.Error(t, err)
		assert.Nil(t, project)
		assert.Equal(t, 404, resp.Response.StatusCode)
	})
}
