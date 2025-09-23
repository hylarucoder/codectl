import React from "react";
import { createRoot } from "react-dom/client";
import { RouterProvider } from "react-router";
import { StyleProvider } from "@ant-design/cssinjs";
import { ConfigProvider } from "antd";
import router from "./router";
// Styles are loaded via index.html -> /src/style.css

createRoot(document.getElementById("root")!).render(
  <StyleProvider layer>
    <ConfigProvider>
      <RouterProvider router={router} />
    </ConfigProvider>
  </StyleProvider>,
);
