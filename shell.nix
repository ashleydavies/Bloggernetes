let
  nixpkgs = fetchTarball "https://github.com/NixOS/nixpkgs/tarball/nixos-24.11";
  pkgs = import nixpkgs { config = {}; overlays = []; };

  nixpkgs-05 = fetchTarball "https://github.com/NixOS/nixpkgs/tarball/nixos-24.05";
  pkgs-05 = import nixpkgs-05 { config = {}; overlays = []; };

  bazel = pkgs.bazel_7;

  go = pkgs.writeScriptBin "go" ''
    #!/usr/bin/env bash
    # Intercept calls to `go` and redirect them to `bazel run @rules_go//go`
    exec "${bazel}/bin/bazel" run @rules_go//go -- "$@"
  '';
in

# For whatever reason, attempting to use 24.11 on macOS results in build-time issues along the lines of
# https://github.com/clangd/clangd/issues/1004.
# Using 24.05 works fine at the moment.
pkgs-05.mkShellNoCC {
  packages = with pkgs; [
    atlas
    bazel
    bazel-buildtools
    go
    gorm-gentool
    kubectl
    k9s
    sqlite
  ];
}
