export const ProductLink = ({ product, href, title, children }) => {
  const productNames = {
    compute: "Compute",
    "api-management": "API Management",
    platform: "Platform",
  };
  return (
    <div className="card unkey-product-card" data-card-href={href}>
      <div data-component-part="card-content-container">
        <h2 data-component-part="card-title">
          <a className="unkey-product-card-title" href={href}>
            {title}
          </a>
        </h2>
        <div data-component-part="card-content">
          <strong>{productNames[product]} docs.</strong> {children}
        </div>
      </div>
    </div>
  );
};
