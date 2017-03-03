package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"bytes"
	"errors"

	"github.com/spf13/cobra"
)

var (
	allImages             = getAllImages()
	allContainers         = getContainers("docker ps -a")
	runningContainers     = getContainers("docker ps")
	stopRunningContainers = true
	printVersion          = false

	RootCmd = &cobra.Command{
		Use:   "dockerrmi [images]",
		Short: "dockerrmi is a handy to remove docker images",
		Long:  `a handy tool to delete user specified docker images along with related containers which use these images`,
		Run: func(cmd *cobra.Command, args []string) {
			run(args)
		},
	}
)

func init() {
	RootCmd.PersistentFlags().BoolVarP(&stopRunningContainers,
		"stoprunningcontainers",
		"s", true,
		"whether or not stop those running containers that use specified image")

	RootCmd.PersistentFlags().BoolVarP(&printVersion,
		"version",
		"v", false,
		"print version of dockerrmi")
}

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

func run(args []string) {
	if printVersion {
		fmt.Println("dockerrmi 0.1")
		return
	}
	if len(args) == 0 {
		pErrorln("you must specify at least one image")
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
				pErrorln("invalid image format, must conform to \"repo:tag\"")
			}
		} else {
			// two cases when image doesn't contain ":"
			// delete image by id
			// delete image by repo but tag ignored
			// so just assume user wanna try these two cases, if more than one image are found, there will be an error
			imageRepo = image
			imageTag = "latest"
			imageID = image
		}
		di, err := getImage(imageRepo, imageTag, imageID)
		if err != nil {
			pInfof("Please provide correct image information, %v\n", err)
		} else {
			if r := getRunningContainers(di); len(r) != 0 {
				if stopRunningContainers {
					for _, c := range r {
						stopC(c.ID)
					}
				} else {
					pInfoln("Running containers exist, please stop these containers manually:")
					for _, c := range r {
						pInfof("\t%s\n", c.ID)
					}
					pInfoln("or try again with \"-r\" flag")
					continue

				}

			}
			removeContainers(di)
			removeImage(di)
		}

	}
}
func getRunningContainers(image *DockerImage) []*DockerContainer {
	rs := []*DockerContainer{}
	for _, c := range runningContainers {
		if c.ImageRepo == image.Repo && c.ImageTag == image.Tag {
			rs = append(rs, c)
		}
	}
	return rs
}
func stopC(containerID string) {
	cmd := `docker stop ` + containerID
	outputBytes, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		pErrorf("failed to stop container, cmd:%s, stdout:%s, err:%s\n",
			cmd,
			string(outputBytes),
			err,
		)
	} else {
		pInfof("stopped container %s\n", containerID)
	}
}
func removeContainers(image *DockerImage) {

	for _, c := range allContainers {
		if c.ImageRepo == image.Repo && c.ImageTag == image.Tag {
			cmd := `docker rm ` + c.ID
			outputBytes, err := exec.Command("sh", "-c", cmd).Output()
			if err != nil {
				pErrorf("failed to remove docker container, cmd:%s, stdout:%s, err:%s\n",
					cmd,
					string(outputBytes),
					err,
				)
			} else {
				pInfof("removed container %s\n", c.ID)
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
		pErrorf("failed to remove image, cmd:%s, stdout:%s, err:%s\n",
			cmd,
			string(outputBytes),
			err,
		)
	} else {
		pInfof("removed image %s\n", image)
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
		var errStr bytes.Buffer
		errStr.WriteString("multiple images found:")
		for _, img := range found {
			errStr.WriteString(fmt.Sprintf("\tid: %s, repo: %s, tag:%s\n", img.ID, img.Repo, img.Tag))
		}
		return nil, errors.New(errStr.String())
	} else if len(found) == 0 {
		return nil, fmt.Errorf(" image not found according to id: %s, repo: %s, tag:%s\n", imageID, repo, tag)
	} else {
		return found[0], nil
	}

}
func getContainers(cmd string) []*DockerContainer {
	all := []*DockerContainer{}
	cmd = cmd + ` --format "{{.ID}}|{{.Image}}"`
	outputBytes, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		pErrorf("failed to execute cmd:%s, stdout:%s, err:%s\n",
			cmd,
			string(outputBytes),
			err,
		)
	}
	scanner := bufio.NewScanner(strings.NewReader(string(outputBytes)))
	for scanner.Scan() {
		id := ""
		imageRepo := ""
		imageTag := ""
		text := scanner.Text()
		if err := scanner.Err(); err != nil {
			pErrorf("failed to read docker image command's output line by line, err:%s\n", err)
		}
		parts := strings.Split(text, "|")
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

		/*		pInfof("container info, id:%s, image repo:%s, image tag:%s\n",
				container.ID,
				container.ImageRepo,
				container.ImageTag,
			)*/
		all = append(all, container)
	}

	return all
}
func getAllImages() []*DockerImage {
	all := []*DockerImage{}
	cmd := `docker images --format "{{.ID}}:{{.Repository}}:{{.Tag}}"`
	outputBytes, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		pErrorf("failed to get docker images, cmd:%s, stdout:%s, err:%s\n",
			cmd,
			string(outputBytes),
			err,
		)
	}
	scanner := bufio.NewScanner(strings.NewReader(string(outputBytes)))
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), ":")
		if len(parts) != 3 {
			continue
		}
		img := &DockerImage{ID: parts[0], Repo: parts[1], Tag: parts[2]}
		all = append(all, img)
	}
	if err := scanner.Err(); err != nil {
		pErrorf("failed to read docker image command's output line by line, err:%s\n", err)
	}
	return all
}
func pErrorln(a ...interface{}) (n int, err error) {
	s := make([]interface{}, len(a)+1)
	s[0] = "[Error]"
	for i, e := range a {
		s[i+1] = e
	}
	return fmt.Fprintln(os.Stderr, s...)
}
func pInfoln(a ...interface{}) (n int, err error) {
	s := make([]interface{}, len(a)+1)
	s[0] = "[Info]"
	for i, e := range a {
		s[i+1] = e
	}
	return fmt.Fprintln(os.Stdout, s...)
}
func pErrorf(format string, a ...interface{}) (n int, err error) {
	return fmt.Fprintf(os.Stderr, "[Error] "+format, a...)
}
func pInfof(format string, a ...interface{}) (n int, err error) {
	return fmt.Fprintf(os.Stdout, "[Info] "+format, a...)
}
