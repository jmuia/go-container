# Testing

I'm learning a ton of new things from this project: namespaces and cgroups; concept of layering file systems; general Linux systems things; and Go!

Adding a test suite is too much "new" (at least, for now).

Instead, I'm going to document some of the manual tests I did to see my code was working as expected.

In the future I might automate these.

## Reexec
Added some print statements showing the PID of the parent and child processes.

```
$ go run container.go
Hello, I am main with pid 8111
Hello, I am container with pid 8115
I am exec
```

## Mount Namespace

### Namespace
I can compare the mount namespace inside and outside the container:
```
vagrant@ubuntu-xenial$ ls -lh /proc/self/ns/mnt
lrwxrwxrwx 1 vagrant vagrant 0 Sep 12 03:02 /proc/self/ns/mnt -> mnt:[4026531840]
```
```
vagrant@ubuntu-xenial$ sudo ./go-container
Hello, I am main with pid 8153
Hello, I am container with pid 8157
root@ubuntu-xenial# ls -lh /proc/self/ns/mnt
lrwxrwxrwx 1 root root 0 Sep 12 03:02 /proc/self/ns/mnt -> mnt:[4026532129]
```

### Private mounts
```
vagrant@ubuntu-xenial$ sudo ./go-container
Hello, I am main with pid 8279
Hello, I am container with pid 8283
root@ubuntu-xenial# mkdir /mnt/iamprivate
root@ubuntu-xenial# mount -t tmpfs tmpfs /mnt/iamprivate
root@ubuntu-xenial# grep iamprivate /proc/mounts
tmpfs /mnt/iamprivate tmpfs rw,relatime 0 0
```

In another process outside the container:
```
vagrant@ubuntu-xenial$ grep iamprivate /proc/mounts
vagrant@ubuntu-xenial$
```

