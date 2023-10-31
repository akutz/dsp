# Datasets Proxy

This project provides a way to support the `vmtoolsd` program's Datasets functionality on VMware Fusion and Workstation by acting as a datasets proxy (`dsp`) for guest variables.


## Install

1. Locate the program `vmtoolsd` inside of a virtual machine (VM), ex. `/bin/vmtoolsd`.
2. Rename `vmtoolsd` so it is still in the same directory, but now has the suffix `.bin`, ex:
    ```shell
    sudo mv /bin/vmtoolsd /bin/vmtoolsd.bin
    ```
3. Download the `dsp` binary for the VM's operating system (OS) from the [releases page](https://github.com/akutz/dsp/releases) so it replaces the now renamed `vmtoolsd`, ex:
    ```shell
    curl -sSL https://github.com/akutz/dsp/releases/download/v0.1.0/govc_Linux_arm64.tar.gz | \
      sudo tar -C /bin --transform='s~dsp~vmtoolsd~' -xzf -
    ```


## Configuration

* The program expects the actual `vmtoolsd` to be located in the same directory as the fake one, but with the `.bin` extension.
* The environment variable `DSP_VMTOOLSD` may be used to specify the absolute path to the real `vmtoolsd` binary, bypassing the logic outlined in the previous step.
* The environment variable `DSP_DEBUG` may be set to a truth-y value to cause the fake `vmtoolsd` to print information helpful to debugging issues.


## Examples


### Setting a single entry

```shell
vmtoolsd --cmd "datasets-set-entry {\"dataset\":\"dstest\", \"entries\":[{\"key\":\"key1\", \"value\":\"val1\"}]}"
```


### Getting a single entry

```shell
$ vmtoolsd --cmd "datasets-get-entry {\"dataset\":\"dstest\", \"keys\":[\"key1\"]}"         
"val1"
```


### Setting multiple entries

```shell
vmtoolsd --cmd "datasets-set-entry {\"dataset\":\"dstest\", \"entries\":[{\"key\":\"key2\", \"value\":\"val2\"},{\"key\":\"key3\", \"value\":\"val3\"}]}"
```


### Getting multiple entries

```shell
$ vmtoolsd --cmd "datasets-get-entry {\"dataset\":\"dstest\", \"keys\":[\"key1\",\"key2\",\"key3\"]}"
"val1"
"val2"
"val3"
```

## Notes

* When there is one or fewer dataset entries to get/set, and when run on a non-Windows platform, `syscall.Exec` is used to call the real `vmtoolsd` program, replacing the running, fake `vmtoolsd` process with the real one.
* When there is more than one dataset entries to get/set or when run on a Windows platform, `os/exec` is used to invoke the real `vmtoolsd` program.
