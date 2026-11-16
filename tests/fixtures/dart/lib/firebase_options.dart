import 'dart:io';

import 'package:firebase_core/firebase_core.dart';

class DefaultFirebaseOptions {
  static FirebaseOptions get currentPlatform =>
      Platform.isIOS ? ios : android;

  static const FirebaseOptions android = FirebaseOptions(
    apiKey: 'AIzaSy' 'Dx3FixtureOnlyKeyNotReal00000000',
    appId: '1:401266817752:android:8c1f0b2e9d4a7c61',
    messagingSenderId: '401266817752',
    projectId: 'pay-wallet-prod',
    storageBucket: 'pay-wallet-prod.appspot.com',
  );

  static const FirebaseOptions ios = FirebaseOptions(
    apiKey: 'AIzaSy' 'Bq7FixtureOnlyKeyNotReal11111111',
    appId: '1:401266817752:ios:2f6e9a0c4b1d8e37',
    messagingSenderId: '401266817752',
    projectId: 'pay-wallet-prod',
    storageBucket: 'pay-wallet-prod.appspot.com',
    iosBundleId: 'vn.paywallet.app',
  );
}
