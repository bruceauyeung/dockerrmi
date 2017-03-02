package main

import (
	"bufio"
	"flag"
	"fmt"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

type DockerImage struct {
	Repo string
	Tag  string
	ID   string
}
type DockerContainer struct {
	ID        string
	ImageRepo string
	ImageTag  string
}

var (
	logger, _     = zap.NewProduction()
	allImages     = getAllImages()
	allContainers = getAllContainers()
)

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		logger.Error("you must specify at least one image")
	}

	for _, image := range args {
		imageRepo := ""
		imageTag := ""
		imageID := ""
		if index := strings.Index(image, ":"); index >= 0 {
			if index > 0 && index < len(image) {
				sepIndex := strings.Index(image, ":")
				imageRepo = image[0:sepIndex]
				imageTag = image[(sepIndex + 1):]

			} else {
				logger.Error("invalid image format, must conform to \"repo:tag\"")
			}
		} else {
			imageID = image
		}
		di, err := getImage(imageRepo, imageTag, imageID)
		if err != nil {
			logger.Info("Please provide correct image info",
				zap.Error(err),
			)
		} else {
			logger.Debug("image exists",
				zap.String("repo", di.Repo),
				zap.String("tag", di.Tag),
				zap.String("ID", di.ID),
			)
			stopContainers(di)
			removeContainers(di)
			removeImage(di)
		}

	}
}
func stopContainers(image *DockerImage) {
	for _, c := range allContainers {
		if c.ImageRepo == image.Repo && c.ImageTag == image.Tag {
			cmd := `docker stop ` + c.ID
			outputBytes, err := exec.Command("sh", "-c", cmd).Output()
			if err != nil {
				logger.Error("failed to stop container",
					zap.String("cmd", cmd),
					zap.String("stdout", string(outputBytes)),
					zap.Error(err),
				)
			} else {
				logger.Info("stopped container",
					zap.String("container id", c.ID),
				)
			}
		}
	}

}
func removeContainers(image *DockerImage) {

	for _, c := range allContainers {
		if c.ImageRepo == image.Repo && c.ImageTag == image.Tag {
			cmd := `docker rm ` + c.ID
			outputBytes, err := exec.Command("sh", "-c", cmd).Output()
			if err != nil {
				logger.Error("failed to remove docker container",
					zap.String("cmd", cmd),
					zap.String("stdout", string(outputBytes)),
					zap.Error(err),
				)
			} else {
				logger.Info("removed container",
					zap.String("container id", c.ID),
				)
			}
		}
	}

}
func removeImage(di *DockerImage) {
	id := di.ID
	repo := di.Repo
	tag := di.Tag
	image := ""
	if id != "" {
		image = id
	} else {
		image = fmt.Sprintf("%s:%s", repo, tag)
	}
	cmd := `docker rmi ` + image
	outputBytes, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		logger.Error("failed to remove docker image",
			zap.String("cmd", cmd),
			zap.String("stdout", string(outputBytes)),
			zap.Error(err),
		)
	} else {
		logger.Info("removed image",
			zap.String("container id", image),
		)
	}
}
func getImage(repo, tag, imageID string) (*DockerImage, error) {
	found := []*DockerImage{}
	for _, img := range allImages {
		if (img.Repo == repo && img.Tag == tag) || (imageID != "" && strings.HasPrefix(img.ID, imageID)) {
			found = append(found, img)
		}
	}
	if len(found) > 1 {
		return nil, fmt.Errorf("multiple images found according to id: %s, repo: %s, tag:%s", imageID, repo, tag)
	} else if len(found) == 0 {
		return nil, fmt.Errorf(" image not found according to id: %s, repo: %s, tag:%s", imageID, repo, tag)
	} else {
		return found[0], nil
	}

}
func getAllContainers() []*DockerContainer {
	all := []*DockerContainer{}
	cmd := `docker ps -a --format "{{.ID}} {{.Image}}"`
	outputBytes, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		logger.Error("failed to execute docker ps",
			zap.String("cmd", cmd),
			zap.String("stdout", string(outputBytes)),
			zap.Error(err),
		)
	}
	scanner := bufio.NewScanner(strings.NewReader(string(outputBytes)))
	for scanner.Scan() {
		id := ""
		imageRepo := ""
		imageTag := ""
		parts := strings.Split(scanner.Text(), " ")
		if len(parts) != 2 {
			continue
		}
		id = parts[0]
		imgParts := strings.Split(parts[1], ":")
		if len(imgParts) == 2 { // repo:tag
			imageRepo = imgParts[0]
			imageTag = imgParts[1]
		} else if len(imgParts) == 1 {
			if img, err := getImage("", "", imgParts[0]); err == nil {
				// docker ps outputs image id in IMAGE column
				// treat imgParts[0] as image id and try to find an image
				imageRepo = img.Repo
				imageTag = img.Tag
			} else if img, err := getImage(imgParts[0], "latest", ""); err == nil {
				// docker ps outputs image repo and ignores "latest" tag
				// treat imgParts[0] as image repo and assume tag is "latest"
				imageRepo = img.Repo
				imageTag = img.Tag
			}
		}
		container := &DockerContainer{ID: id, ImageRepo: imageRepo, ImageTag: imageTag}
		logger.Debug("container info",
			zap.String("container ID", container.ID),
			zap.String("image repo", container.ImageRepo),
			zap.String("image tag", container.ImageTag),
		)
		all = append(all, container)
	}
	if err := scanner.Err(); err != nil {
		logger.Error("failed to read docker image command's output line by line",
			zap.Error(err))
	}
	return all
}
func getAllImages() []*DockerImage {
	all := []*DockerImage{}
	cmd := `docker images --format "{{.ID}}:{{.Repository}}:{{.Tag}}"`
	outputBytes, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		logger.Error("failed to execute docker images",
			zap.String("cmd", cmd),
			zap.String("stdout", string(outputBytes)),
			zap.Error(err),
		)
	}
	scanner := bufio.NewScanner(strings.NewReader(string(outputBytes)))
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), ":")
		if len(parts) != 3 {
			continue
		}
		img := &DockerImage{ID: parts[0], Repo: parts[1], Tag: parts[2]}
		logger.Debug("image info",
			zap.String("ID", img.ID),
			zap.String("repo", img.Repo),
			zap.String("tag", img.Tag),
		)
		all = append(all, img)
	}
	if err := scanner.Err(); err != nil {
		logger.Error("failed to read docker image command's output line by line",
			zap.Error(err))
	}
	return all
}
