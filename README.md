
docker execution

```sh
docker run --rm -it -v (pwd):/app -v /tmp/dvc:/tmp/dvc dvc-uploader --repo https://metalblueberry:<your github token>@github.com/MetalBlueberry/dvc-uploader-test test.txt
```
