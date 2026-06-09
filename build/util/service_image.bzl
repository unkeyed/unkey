load("@rules_oci//oci:defs.bzl", "oci_image", "oci_image_index", "oci_load", "oci_push")
load("@rules_pkg//pkg:tar.bzl", "pkg_tar")

def service_image(name, repository, local_tag, description, binary_name = None):
    """Defines release and local-dev OCI image targets for one service.

    Must be called from the same Bazel package that defines the service's
    go_binary targets. The macro expects two binaries named `<name>_linux_amd64`
    and `<name>_linux_arm64`, each with `out = "bin/<binary_name>"`.

    Args:
        name: Bazel target prefix; also defaults to the in-image binary name.
        repository: GHCR image repository (e.g. "ghcr.io/unkeyed/api").
        local_tag: Docker tag applied by the `load` target (e.g. "unkey/api:dev").
        description: Value for the OCI image description label.
        binary_name: Filename of the binary inside the image. Defaults to `name`.
            Pass explicitly when the binary uses a different naming convention
            than the Bazel target prefix (e.g. "control-api" vs "control_api").

    Generates targets:
        :image_linux_amd64, :image_linux_arm64: per-arch images
        :image:                                 multi-arch index
        :push:                                  oci_push to `repository`
        :load:                                  oci_load tagged `local_tag`
    """
    if binary_name == None:
        binary_name = name

    # strip_prefix is relative to this macro caller's BUILD package. Callers
    # must define the go_binary targets in the same package as this macro
    # invocation; otherwise "bin" won't match and the binary lands at
    # /build/<svc>/bin/<svc> inside the image instead of /<svc>.
    pkg_tar(
        name = name + "_tar_linux_amd64",
        srcs = [":" + name + "_linux_amd64"],
        extension = "tar.gz",
        package_dir = "/",
        strip_prefix = "bin",
    )

    pkg_tar(
        name = name + "_tar_linux_arm64",
        srcs = [":" + name + "_linux_arm64"],
        extension = "tar.gz",
        package_dir = "/",
        strip_prefix = "bin",
    )

    labels = {
        "org.opencontainers.image.description": description,
        "org.opencontainers.image.source": "https://github.com/unkeyed/unkey",
    }

    oci_image(
        name = "image_linux_amd64",
        base = "@distroless_static_linux_amd64",
        entrypoint = ["/" + binary_name],
        labels = labels,
        tars = [":" + name + "_tar_linux_amd64"],
    )

    oci_image(
        name = "image_linux_arm64",
        base = "@distroless_static_linux_arm64_v8",
        entrypoint = ["/" + binary_name],
        labels = labels,
        tars = [":" + name + "_tar_linux_arm64"],
    )

    oci_image_index(
        name = "image",
        images = [
            ":image_linux_amd64",
            ":image_linux_arm64",
        ],
    )

    oci_push(
        name = "push",
        image = ":image",
        repository = repository,
    )

    oci_load(
        name = "load",
        image = select({
            "@platforms//cpu:arm64": ":image_linux_arm64",
            "@platforms//cpu:x86_64": ":image_linux_amd64",
            "//conditions:default": ":image_linux_amd64",
        }),
        repo_tags = [local_tag],
        visibility = ["//visibility:public"],
    )
