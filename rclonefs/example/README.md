### How to pass this test

This test can be used with any cloud storage,
which is available in RClone.

You must have installed RClone on your computer.

### Example: how to run this test with PCloud

1. Create PCloud account.
2. Create rclone configuration using this instruction:

https://rclone.org/pcloud/

Let's presume that the name of your remote storage, which
you put during the first step, is pcloud1.

```
name> pcloud
```

3. Put the file employee.yml or any other to the root
directory of your PCloud storage.
4. Make sure that your configuration works. For example,
run this command:

```
rclone cat pcloud1:employee.yml
```

pcloud1 is the name of your remote storage:
(2. name> pcloud1), person.json is in the root folder.

- you will see the content of the file in your console.

If your cloud storage contains buckets (Amazon S3,
minio, Backblaze B2) and the file employee.yml is put into
the bucket1, the access to the file is:

```go
rclone cat minio1:bucket1/employee.yml
```

minio1 is the name of your cloud storage.

The data is stored in the file rclone.conf. On Linux,
its default path is:

```
${HOME}/.config/rclone/rclone.conf
```

5. Create "/cfg/json" directory in your remote storage. 
It will be your working directory in the example.

6. Put the name of your remote storage with the full
path '/cfg/json' to the file 'cloud.txt'

```
pcloud_mv1:/cfg/json
```

7. Run the example:

```
go run example.go
```
