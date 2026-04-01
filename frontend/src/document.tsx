import { Meta, Title, Links, Main, Scripts } from 'ice';

export default function Document() {
  return (
    <html>
      <head>
        <meta charSet="utf-8" />
        <link rel="icon" href="/logo-ai.svg" type="image/svg+xml" />
        <meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1, user-scalable=no" />
        {/* 内容安全策略 */}
        <meta
          httpEquiv="Content-Security-Policy"
          content={
            "default-src 'self'; " +
            "script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
            "worker-src 'self' blob:; " +
            "style-src 'self' 'unsafe-inline'; " +
            "img-src * data:; " +
            "font-src 'self' data:; " +
            "connect-src 'self'; " +
            "child-src 'self' blob:; " +
            "frame-src *; "
          }
        />
        <Meta />
        <Title />
        <Links />
      </head>
      <body>
        <Main />
        <Scripts />
      </body>
    </html>
  );
}
