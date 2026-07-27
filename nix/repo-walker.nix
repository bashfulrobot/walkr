# repo-walker: pure Go CLI, no cgo, no frontend build step.
{ lib
, buildGoModule
}:

buildGoModule {
  pname = "repo-walker";
  version = "0.0.1";

  src = lib.cleanSource ../.;

  vendorHash = "sha256-pkEa4Z+MsvVR5M+/QC3sIyl/H1wMDheHJXqRM60q7Yo=";

  env.CGO_ENABLED = "0";

  # main.go lives at the module root; everything under internal/ is a
  # dependency, not a separate entrypoint.
  subPackages = [ "." ];

  ldflags = [ "-w" "-s" ];

  doCheck = false;

  meta = {
    description = "Renders a hand-authored markdown walkthrough into a static site";
    homepage = "https://github.com/bashfulrobot/repo-walker";
    license = lib.licenses.mit;
    mainProgram = "repo-walker";
  };
}
