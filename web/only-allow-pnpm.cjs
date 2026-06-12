// Block npm and yarn from being used in this project.
// This runs as a preinstall script to enforce pnpm usage.
const agent = process.env.npm_config_user_agent || "";
if (!agent.startsWith("pnpm")) {
  console.error(
    "\n\x1b[31mERROR: Use pnpm to install dependencies in this project.\x1b[0m\n" +
    "       Install it via: npm i -g pnpm\n" +
    "       Then run: pnpm install\n"
  );
  process.exit(1);
}
