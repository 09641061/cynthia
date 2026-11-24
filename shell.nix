{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  packages = [
    pkgs.go
    pkgs.gnumake
    pkgs.stdenv.cc
  ];

  shellHook = ''
    echo "Ready. Run: make run"
  '';
}
