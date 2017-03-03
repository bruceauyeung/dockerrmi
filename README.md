# dockerrmi
a handy tool to delete user specified docker images along with related containers which use these images

## what `dockerrmi` does
`dockerrmi` is similar with `docker rmi`. It works following these steps:

1. determines if there are running containers which are using specified images,
2. if true, try to stop these running containers by default, after all these containers are stopped,
3. try to remove all containers which are using specified images,
4. remove all specified images

## differences between `dockerrmi` and `docker rmi -f`

1. if there are running/stopped containers which are using specified images, `dockerrmi` will stop/remove those containers first, 
but `docker rmi -f` doesn't.
2. if there are running containers which are using specified images, `dockerrmi` will not result in dangling images, 
but `docker rmi -f` does because it doesn't stop running containers

## installation

If you have Go installed, you can do:

1. go get -u -v github.com/spf13/cobra/cobra
2. go get -u -v github.com/spf13/viper
3. go get -u -v github.com/bruceauyeung/dockerrmi

else you just need to download https://github.com/bruceauyeung/dockerrmi/raw/master/dockerrmi-linux-amd64, rename it to what you want and put it in `$PATH`
## usage

~~~~
# dockerrmi --help
a handy tool to delete user specified docker images along with related containers which use these images

Usage:
  dockerrmi [images] [flags]

Flags:
  -s, --stoprunningcontainers   whether or not stop those running containers that use specified image (default true)
  -v, --version                 print version of dockerrmi

# dockerrmi busybox:latest
[Info] removed image 2b8fd9751c4c

~~~~
