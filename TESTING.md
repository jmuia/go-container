# Testing

I'm learning a ton of new things from this project: namespaces and cgroups; concept of layering file systems; general Linux systems things; and Go!

Adding a test suite is too much "new" (at least, for now).

Instead, I'm going to document some of the manual tests I did to see my code was working as expected.

In the future I might automate these.

/Note from the future: this project has grown enough in complexity I would feel much more comfortable if I had test suite to run/

## Reexec
Added some print statements showing the PID of the parent and child processes.

```
$ go run container.go
Hello, I am main with pid 8111
Hello, I am container with pid 8115
I am exec
```

## Mount namespace

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

## Images and containers
Run the container and check that there is a copy of the image. This will get better.

```
vagrant@ubuntu-xenial$ ls containers
vagrant@ubuntu-xenial$ sudo ./go-container
Hello, I am main with pid 20991
Hello, I am container with pid 20996
root@ubuntu-xenial# ls containers/e01a04e8-6e30-4d5a-a99c-a722b02bad04/rootfs/
bin  dev  etc  home  lib  media  mnt  proc  root  run  sbin  srv  sys  tmp  usr  var
```

## Pivot root
We had to change our shell to `/bin/sh` because alpine doesn't have bash. We can see we have a new view of the file system now.

Now there's nothing in some file systems like /proc and /sys, so we can't see our hosts mounts that way anymore.
```
vagrant@ubuntu-xenial$ sudo ./go-container
Hello, I am main with pid 21656
Hello, I am container with pid 21661
/ # ls
bin    dev    etc    home   lib    media  mnt    proc   root   run    sbin   srv    sys    tmp    usr    var
/ # mount
mount: no /proc/mounts
/ # pwd
/
/ # exit
```

## Mount special file systems
We can see interesting things in our special filesystems now and devtmpfs has created a bunch of devices for us.

/Note from the future, using a separate user namespace (and not using sudo to run ./go-container) we've seemingly lost the ability to mount devtmpfs. This doesn't totally surprise me. Will have to add devices manually for now and mount tmpfs on /dev. This seems to indicate that devtmpfs does not set `FS_USERNS_MOUNT` flag./

```
vagrant@ubuntu-xenial$ sudo ./go-container
Hello, I am main with pid 22987
Hello, I am container with pid 22992
/ # ls /sys
block       class       devices     fs          kernel      power
bus         dev         firmware    hypervisor  module
/ # mount
home_vagrant_go_src_github.com_jmuia_go-container on / type vboxsf (rw,nodev,relatime)
proc on /proc type proc (rw,nosuid,nodev,noexec,relatime)
sysfs on /sys type sysfs (rw,nosuid,nodev,noexec,relatime)
devtmpfs on /dev type devtmpfs (rw,nosuid,relatime,size=498876k,nr_inodes=124719,mode=755)
devpts on /dev/pts type devpts (rw,nosuid,noexec,relatime,mode=600,ptmxmode=000)
tmpfs on /dev/shm type tmpfs (rw,nosuid,nodev,relatime)
/ # ls /dev
autofs              mapper              tty0                tty36               tty63               ttyS4
block               mcelog              tty1                tty37               tty7                ttyS5
...
loop6               stdout              tty34               tty61               ttyS30              zero
loop7               tty                 tty35               tty62               ttyS31
/ # ps | head
PID   USER     TIME  COMMAND
    1 root      0:06 {systemd} /sbin/init
    2 root      0:00 [kthreadd]
    3 root      0:00 [ksoftirqd/0]
    5 root      0:00 [kworker/0:0H]
    7 root      0:01 [rcu_sched]
    8 root      0:00 [rcu_bh]
    9 root      0:00 [migration/0]
   10 root      0:00 [watchdog/0]
   11 root      0:00 [watchdog/1]
```

We still see host processes, but we'll deal with that later.

We also see `home_vagrant_go_src_github.com_jmuia_go-container on / type vboxsf (rw,nodev,relatime)` in /proc/mounts.
That's a host mount? I happen to be using a Vagrant VM for testing, and I have the golang workspace setup as a shared folder.

If instead I mount a `tmpfs` to get around the `pivot_root` requirement:
```
// bind mount containerRoot to itself to circumvent pivot_root requirement.
if err := syscall.Mount("tmpfs", containerRoot, "tmpfs", 0, ""); err != nil {
    panic(fmt.Sprintf("Error changing root file system (mount tmpfs containerRoot): %s\n", err))
}
```
I see `tmpfs on / type tmpfs (rw,relatime)` instead.

I think I get it. `home_vagrant_go_src_github.com_jmuia_go-container` is just the "device" that's mounted. So even though in the call to `mount(2)` the source "device" is `containerRoot`, the actual bind mount will have the same device as the original. That makes sense when you consider what bind mounts actually accomplish.

We'll come back to making this better.


## UTS namespace
I've set the hostname to the container id (and updated the PS1, for fun). Since we're in a new UTS namespace, the host won't be affected.
```
vagrant@ubuntu-xenial$ sudo ./go-container
Hello, I am main with pid 3727
Hello, I am container with pid 3732
root@76e03801-23ed-4c71-a1ba-c47f94811d0d$ hostname
76e03801-23ed-4c71-a1ba-c47f94811d0d
root@76e03801-23ed-4c71-a1ba-c47f94811d0d$ hostname container
root@76e03801-23ed-4c71-a1ba-c47f94811d0d$ hostname
container
root@76e03801-23ed-4c71-a1ba-c47f94811d0d$ exit
vagrant@ubuntu-xenial$ hostname
ubuntu-xenial
```
We can also compare the inode number in `/proc/self/ns/uts` for the container and the host.

## PID namespace

Before the namespace, we can see all the host processes:
```
vagrant@ubuntu-xenial$ make run
go build
sudo ./go-container
Hello, I am main with pid 4301
Hello, I am container with pid 4306
root@42456060-cfcc-44e3-b81b-dcaf38a87865$ ps
PID   USER     TIME  COMMAND
    1 root      0:03 {systemd} /sbin/init
    ...
 4205 1000      0:00 sshd: vagrant@pts/1
 4206 1000      0:00 -bash
 4223 root      0:00 [kworker/u4:1]
 4268 1000      0:00 top
 4269 1000      0:00 make run
 4300 root      0:00 sudo ./go-container
 4301 root      0:00 ./go-container
 4306 root      0:00 {exe} container
 4311 root      0:00 /bin/sh
 4312 root      0:00 ps
 ```

And we can kill them!
```
root@42456060-cfcc-44e3-b81b-dcaf38a87865$ ps | grep top
 4268 1000      0:00 top
 4314 root      0:00 grep top

root@42456060-cfcc-44e3-b81b-dcaf38a87865$ kill -9 4268

root@42456060-cfcc-44e3-b81b-dcaf38a87865$ ps | grep top
 4316 root      0:00 grep top
```

Once we enter the namespace, we can no longer see the host processes:
```
root@38f03563-6500-4ef5-b583-087851c9fdad$ ps
PID   USER     TIME  COMMAND
    1 root      0:00 {exe} container
    6 root      0:00 /bin/sh
    7 root      0:00 ps
```

Yikes, there's a bug. We really want to be using `syscall.Exec` rather Go's `exec.Command`.

```
root@d349ced3-db1d-4b21-911f-d6d141534b4c$ ps
PID   USER     TIME  COMMAND
    1 root      0:00 sh
    6 root      0:00 ps
```

Much better.

## User namespace

Now we can run ./go-container without root. We've set up UID and GID mapping so that inside the container we seem to be root.

Before (ran from the parent namespace; 7564 is the container) we were actually root:
```
vagrant@ubuntu-xenial:~$ cat /proc/7564/uid_map
         0          0 4294967295
```

After (ran from the parent namespace; 7666 is the container) we're still user 1000:
```
vagrant@ubuntu-xenial:~$ cat /proc/7666/uid_map
         0       1000          1
```

I had some troubles getting this to work. I had to stop using `devtmpfs` and instead mount `tmpfs` at `/dev`. I also had some funky permissions stuff when I tried creating containers in the default `./containers` directory. Even though `vagrant` user owns the directory there was something odd going on with it being a `vboxfs` mount. I can just create them at `~/containers` for now.

/Note from the future: I've removed the user namespace and uid/gid mapping. It's considered an advanced feature of Docker and its interactions with other namespaces make things more complicated. We'll keep using `sudo` to run our container for now. We can `setuid` to downgrade our privileges in the container. We can maybe use the user namespace in tandem with `setuid`, entering the namespace after we've done all the setup with root access./

## Net namespace
Adding a net namespace is easy if you don't set up any devices after!

We can check the output of `ip a` before and after.

Before we can see the hosts devices.

```
root@0885a90e-b2fb-4627-b5a6-679a1ac2ff01$ ip a
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN qlen 1
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host 
       valid_lft forever preferred_lft forever
2: enp0s3: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc pfifo_fast state UP qlen 1000
    link/ether 02:48:e1:e3:74:ed brd ff:ff:ff:ff:ff:ff
    inet 10.0.2.15/24 brd 10.0.2.255 scope global enp0s3
       valid_lft forever preferred_lft forever
    inet6 fe80::48:e1ff:fee3:74ed/64 scope link 
       valid_lft forever preferred_lft forever
3: brg0: <NO-CARRIER,BROADCAST,MULTICAST,UP> mtu 1500 qdisc noqueue state DOWN 
    link/ether 00:00:00:00:00:00 brd ff:ff:ff:ff:ff:ff
    inet 10.10.10.1/24 scope global brg0
       valid_lft forever preferred_lft forever
    inet6 fe80::cce5:86ff:fe25:70ac/64 scope link 
       valid_lft forever preferred_lft forever
```

After using a new net namespace we just have `lo` and it's down:
```
root@12140b19-7415-4fa9-9aaf-ce91836f4499$ ip a
1: lo: <LOOPBACK> mtu 65536 qdisc noop state DOWN qlen 1
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
```

## Devices

The auto-magic of `devtmpfs` is great, but it seems to me that it's leaking the host devices.

```
root@c2f51e5d-c642-4cf9-8901-101f0157dd2f$ ls /dev/
autofs              mapper              tty0                tty36               tty63               ttyS4
block               mcelog              tty1                tty37               tty7                ttyS5
bsg                 mem                 tty10               tty38               tty8                ttyS6
btrfs-control       memory_bandwidth    tty11               tty39               tty9                ttyS7
char                mqueue              tty12               tty4                ttyS0               ttyS8
console             net                 tty13               tty40               ttyS1               ttyS9
core                network_latency     tty14               tty41               ttyS10              ttyprintk
cpu_dma_latency     network_throughput  tty15               tty42               ttyS11              uinput
disk                null                tty16               tty43               ttyS12              urandom
ecryptfs            port                tty17               tty44               ttyS13              vboxguest
fd                  ppp                 tty18               tty45               ttyS14              vboxuser
full                psaux               tty19               tty46               ttyS15              vcs
fuse                ptmx                tty2                tty47               ttyS16              vcs1
hpet                pts                 tty20               tty48               ttyS17              vcs2
hugepages           random              tty21               tty49               ttyS18              vcs3
hwrng               rfkill              tty22               tty5                ttyS19              vcs4
initctl             rtc                 tty23               tty50               ttyS2               vcs5
input               rtc0                tty24               tty51               ttyS20              vcs6
kmsg                sda                 tty25               tty52               ttyS21              vcsa
lightnvm            sda1                tty26               tty53               ttyS22              vcsa1
log                 sdb                 tty27               tty54               ttyS23              vcsa2
loop-control        sg0                 tty28               tty55               ttyS24              vcsa3
loop0               sg1                 tty29               tty56               ttyS25              vcsa4
loop1               shm                 tty3                tty57               ttyS26              vcsa5
loop2               snapshot            tty30               tty58               ttyS27              vcsa6
loop3               snd                 tty31               tty59               ttyS28              vfio
loop4               stderr              tty32               tty6                ttyS29              vga_arbiter
loop5               stdin               tty33               tty60               ttyS3               vhost-net
loop6               stdout              tty34               tty61               ttyS30              zero
loop7               tty                 tty35               tty62               ttyS31
```

We're going to replace it with `tmpfs` and mount some devices ourselves.

```
root@21782b9e-1718-40fe-8f1e-6bf2e60be55d$ ls /dev
console  full     ptmx     random   stderr   stdout   urandom
fd       null     pts      shm      stdin    tty      zero
```
