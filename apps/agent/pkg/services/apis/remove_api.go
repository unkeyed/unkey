package apis

import (
	"context"
	"fmt"
)

func (s *service) RemoveApi(ctx context.Context, req RemoveApiRequest) (RemoveApiResponse, error) {

	err := s.database.DeleteApi(ctx, req.ApiId)
	if err != nil {
		return RemoveApiResponse{}, fmt.Errorf("unable to delete api: %w", err)
	}
	return RemoveApiResponse{}, nil
}
