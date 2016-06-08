package serverrpc

import (
	"bytes"
	"io"

	enginetypes "github.com/docker/engine-api/types"
	"github.com/golang/glog"
	"github.com/hyperhq/hyperd/types"
	"golang.org/x/net/context"
)

// ImageList implements GET /images/get
func (s *ServerRPC) ImageList(ctx context.Context, req *types.ImageListRequest) (*types.ImageListResponse, error) {
	glog.V(3).Infof("ImageList with request %s", req.String())

	images, err := s.daemon.Daemon.Images(req.FilterArgs, req.Filter, req.All)
	if err != nil {
		glog.Errorf("ImageList error: %v", err)
		return nil, err
	}

	result := make([]*types.ImageInfo, 0, len(images))
	for _, image := range images {
		result = append(result, &types.ImageInfo{
			Id:          image.ID,
			ParentID:    image.ParentID,
			RepoTags:    image.RepoTags,
			RepoDigests: image.RepoDigests,
			Created:     image.Created,
			VirtualSize: image.VirtualSize,
			Labels:      image.Labels,
		})
	}

	return &types.ImageListResponse{
		ImageList: result,
	}, nil
}

// ImagePull pulls a image from registry
func (s *ServerRPC) ImagePull(req *types.ImagePullRequest, stream types.PublicAPI_ImagePullServer) error {
	glog.V(3).Infof("ImagePull with request %s", req.String())

	authConfig := &enginetypes.AuthConfig{}
	if req.Auth != nil {
		authConfig = &enginetypes.AuthConfig{
			Username:      req.Auth.Username,
			Password:      req.Auth.Password,
			Auth:          req.Auth.Auth,
			Email:         req.Auth.Email,
			ServerAddress: req.Auth.Serveraddress,
			RegistryToken: req.Auth.Registrytoken,
		}
	}

	buffer := bytes.NewBuffer([]byte{})
	exit := make(chan error, 1)
	go func() {
		exit <- s.daemon.CmdImagePull(req.Image, req.Tag, authConfig, nil, buffer)
	}()

	for {
		data := make([]byte, 512)
		n, err := buffer.Read(data)
		if err == io.EOF {
			break
		}

		if err != nil {
			glog.Errorf("Read image pull stream error: %v", err)
			return err
		}

		if err := stream.Send(&types.ImagePullResponse{Data: data[:n]}); err != nil {
			return err
		}
	}

	return <-exit
}

// ImagePush pushes a local image to registry
func (s *ServerRPC) ImagePush(req *types.ImagePushRequest, stream types.PublicAPI_ImagePushServer) error {
	glog.V(3).Infof("ImagePush with request %s", req.String())

	authConfig := &enginetypes.AuthConfig{}
	if req.Auth != nil {
		authConfig = &enginetypes.AuthConfig{
			Username:      req.Auth.Username,
			Password:      req.Auth.Password,
			Auth:          req.Auth.Auth,
			Email:         req.Auth.Email,
			ServerAddress: req.Auth.Serveraddress,
			RegistryToken: req.Auth.Registrytoken,
		}
	}

	buffer := bytes.NewBuffer([]byte{})
	exit := make(chan error, 1)
	go func() {
		exit <- s.daemon.CmdImagePush(req.Repo, req.Tag, authConfig, nil, buffer)
	}()

	for {
		data := make([]byte, 512)
		n, err := buffer.Read(data)
		if err == io.EOF {
			break
		}

		if err != nil {
			glog.Errorf("Read image push stream error: %v", err)
			return err
		}

		if err := stream.Send(&types.ImagePushResponse{Data: data[:n]}); err != nil {
			return err
		}
	}

	return <-exit
}

// ImageDelete deletes a image from hyperd
func (s *ServerRPC) ImageRemove(ctx context.Context, req *types.ImageRemoveRequest) (*types.ImageRemoveResponse, error) {
	glog.V(3).Infof("ImageDelete with request %s", req.String())

	resp, err := s.daemon.CmdImageDelete(req.Image, req.Force, req.Prune)
	if err != nil {
		glog.Errorf("DeleteImage failed: %v")
		return nil, err
	}

	return &types.ImageRemoveResponse{
		Images: resp,
	}, nil
}
