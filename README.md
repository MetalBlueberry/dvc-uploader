# DVC UPLOADER 

> The current status is [POC](https://www.google.com/search?sxsrf=ALeKk01j8l9KxIuWOoGsDhF8zzLXdDI_ZA%3A1605383339007&ei=qjSwX9qCPZGX8gKyiaqACg&q=proof+of+concept&oq=proof+of&gs_lcp=CgZwc3ktYWIQAxgAMgcIABDJAxBDMgQIABBDMgQIABBDMgUIABDLATICCAAyBQgAEMsBMgUIABDLATIFCAAQywEyBQgAEMsBMgUILhDLAToECAAQRzoHCCMQyQMQJzoECCMQJzoECC4QQzoICC4QxwEQowJQsF1YhWdgtW1oAHAEeACAAZsBiAGNB5IBAzEuN5gBAKABAaoBB2d3cy13aXrIAQjAAQE&sclient=psy-ab)

This project aims to create a zero dependency tool to upload/download files tracked under [DVC](https://dvc.org) repositories

The current implementation just uploads a single file to a dvc tracked repository without downloading the entire folder.

## Remotes

currently, only local filesystem is supported as a remote. But I've plans of supporting the same as dvc

[Example here](https://github.com/MetalBlueberry/dvc-uploader-test)

## Run

docker execution

```sh
docker run --rm -it -v (pwd):/app -v /tmp/dvc:/tmp/dvc dvc-uploader --repo https://metalblueberry:<your github token>@github.com/MetalBlueberry/dvc-uploader-test test.txt
```

see dvc-uploader for more details
