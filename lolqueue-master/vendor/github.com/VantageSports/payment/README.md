# payment

This api deals with vantagesports.gg customers making payments for any of our products via Stripe

Method | Endpoint                      | Description                         | Data Format
|------|-------------------------------|-------------------------------------|---------------------------------------------
| POST | /payment/v1/Purchase          | purchase one of our products created in Stripe | { user_id: userId, sku_id: skuId }
| POST | /payment/v1/SavePaymentSource | save a payment source for a UserCustomer.  If the UserCustomer already exists it updates the payment source, if they don't a UserCustomer is created with the payment source | { stripe_token: stripeToken, email: email, user_id: userId}
| POST | /payment/v1/ListPaymentSources| list the payment sources of a UserCustomer | { user_id: userId }
| POST | /payment/v1/ListProducts      | list all the Stripe products, can filter by product_id | { product_id: productId }
