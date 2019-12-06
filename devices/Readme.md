# Devices

This directory contains support tools and popular checks that can be included in FwAnalyzer configs for multiple targets.

- [Android](android)
- [generic Linux](generic)

## Check.py

check.py is a universal script to run FwAnalyzer. It will unpack (with the help of a unpacker; see below) firmware
and run fwanalyzer against each of the target filesystems, it will combine all of the reports
into one big report. In addition it will do some post processing of the filetree files (if present) and
append the result to the report.

Using check.py is straight forward (the example below is for an Android OTA firmware - make sure you have the required Android unpacking tools installed and added to your PATH, see: [Android](android/Readme.md)):

```sh
check.py --unpacker android/unpack.sh --fw some_device_ota.zip --cfg-path android --cfg-include android --fwanalyzer-bin ../build/fwanalyzer
```

The full set of options is described below:
```
usage: check.py [-h] --fw FW --unpacker UNPACKER --cfg-path CFG_PATH
                [--cfg-include-path CFG_INCLUDE_PATH] [--report REPORT]
                [--keep-unpacked] [--fwanalyzer-bin FWANALYZER_BIN]
                [--fwanalyzer-options FWANALYZER_OPTIONS]

optional arguments:
  -h, --help            show this help message and exit
  --fw FW               path to firmware file OR path to unpacked firmware
  --unpacker UNPACKER   path to unpacking script
  --cfg-path CFG_PATH   path to directory containing config files
  --cfg-include-path CFG_INCLUDE_PATH
                        path to config include files
  --report REPORT       report file
  --keep-unpacked       keep unpacked data
  --fwanalyzer-bin FWANALYZER_BIN
                        path to fwanalyzer binary
  --fwanalyzer-options FWANALYZER_OPTIONS
                        options passed to fwanalyzer
```

The _--keep-unpacked_ option will NOT delete the temp directory that contains the unpacked files.
Once you have the unpacked directory you can pass it to the _--fw_ option to avoid unpacking the
firmware for each run (e.g. while you test/modify your configuration files). See the example below.

```sh
check.py --unpacker android/unpack.sh --fw /tmp/tmp987689123 --cfg-path android --cfg-include android --fwanalyzer-bin ../build/fwanalyzer
```

### unpacker

The unpacker is used by check.py to _unpack_ firmware.
The unpacker needs to be an executable file, that takes two parameters first the `file` to unpack
and second the `path to the config files` (the path that was provided via --cfg-path).

The unpacker needs to output a set of targets, the targets map a config file to a filesystem image (or directory).
The targets are specified as a JSON object.

The example below specifies two targets:

- system : use _system.toml_ when analyzing _system.img_
- boot: use _boot.toml_ when analyzing the content of directory _boot/_

```json
{ "system": "system.img" , "boot": "boot/" }
```

See [Android/unpack.sh](android/unpack.sh) for a real world example.
