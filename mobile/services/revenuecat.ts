/**
 * RevenueCat integration for subscription management
 *
 * Package: react-native-purchases
 * Install: npx expo install react-native-purchases
 *
 * Stub implementation — real calls require the SDK + API keys configured
 * in app.json (ios.config / android.config).
 */

export const ENTITLEMENTS = {
  adFree: 'ad_free',
};

export const OFFERINGS = {
  default: 'default',
};

/**
 * Returns true if the user has an active ad-free subscription.
 *
 * TODO: Replace with:
 *   const info = await Purchases.getCustomerInfo();
 *   return info.entitlements.active[ENTITLEMENTS.adFree] !== undefined;
 */
export async function checkSubscriptionStatus(): Promise<boolean> {
  return false;
}

/**
 * Initiates the ad-free purchase flow.
 *
 * TODO: Replace with:
 *   const offerings = await Purchases.getOfferings();
 *   const pkg = offerings.current?.availablePackages[0];
 *   if (!pkg) return false;
 *   const { customerInfo } = await Purchases.purchasePackage(pkg);
 *   return customerInfo.entitlements.active[ENTITLEMENTS.adFree] !== undefined;
 */
export async function purchaseAdFree(): Promise<boolean> {
  return false;
}
