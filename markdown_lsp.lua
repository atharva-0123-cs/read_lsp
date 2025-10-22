
-- ~/.config/nvim/lua/custom/markdown_lsp.lua
local client_id = nil

vim.api.nvim_create_autocmd("FileType", {
  pattern = { "markdown" },
  callback = function()
    -- Prevent multiple clients
    if client_id then return end

    local cmd = { vim.fn.expand("~/read_lsp/main") }

    local id = vim.lsp.start({
      name = "markdown_lsp",
      cmd = cmd,
      root_dir = vim.fn.getcwd(), -- optional, can remove if unnecessary
      on_attach = function(client, bufnr)
        print("✅ Custom Markdown LSP attached:", client.name)
      end,
      filetypes = { "markdown" },
    })

    -- Store client id (not the client object)
    if id then
      client_id = id
    end
  end,
})
