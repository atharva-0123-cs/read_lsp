local function server_command()
  local configured = os.getenv("READ_LSP_BIN")
  if configured and configured ~= "" then
    return { configured }
  end

  local on_path = vim.fn.exepath("read_lsp")
  if on_path ~= "" then
    return { on_path }
  end

  return { vim.fn.expand("~/read_lsp/read_lsp") }
end

vim.api.nvim_create_autocmd("FileType", {
  pattern = { "markdown" },
  callback = function()
    vim.lsp.start({
      name = "read_lsp",
      cmd = server_command(),
      root_dir = vim.fn.getcwd(),
      on_attach = function(client, bufnr)
        vim.notify("read_lsp attached to buffer " .. bufnr, vim.log.levels.INFO)
      end,
    })
  end,
})
