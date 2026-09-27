/** @type {import('steiger').Config} */
import { defineConfig } from "steiger";
import fsd from "@feature-sliced/steiger-plugin";

export default defineConfig({
  plugins: [fsd],
});
