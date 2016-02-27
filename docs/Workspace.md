TODO: #27

# Folder structure

```
├── config
│   ├── bootparams
│   │   ├── main
│   │   └── ...
│   ├── cloudconfig
│   │   ├── main
│   │   └── ...
│   └── ignition
│       ├── main
│       └── ...
├── files
├── images
│   ├── [CoreOS Version (i.e. 899.5.0)]
│   │   ├── coreos_production_pxe_image.cpio.gz
│   │   └── coreos_production_pxe.vmlinuz
│   └── version.txt
└── initial.yaml
```

## Examples

* [Using flags](https://github.com/cafebazaar/blacksmith-workspace-kubernetes/blob/temporary/config/cloudconfig/main)
* [Using api to update flags](https://github.com/cafebazaar/blacksmith-workspace-kubernetes/blob/temporary/config/cloudconfig/partitioning.sh#L11)
